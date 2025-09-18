package conversations

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ageniuscoder/mmchat/backend/internal/auth"
	"github.com/ageniuscoder/mmchat/backend/internal/chat"
	"github.com/ageniuscoder/mmchat/backend/internal/httpx"
	"github.com/ageniuscoder/mmchat/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Service struct {
	DB  *sql.DB
	Hub *chat.Hub
}

type privateReq struct {
	OtherUserId int64 `json:"other_user_id"`
}

type groupReq struct {
	Name      string  `json:"name"`
	MemberIDs []int64 `json:"member_ids"`
}

type addReq struct {
	UserID int64 `json:"user_id"`
}

func Register(rg *gin.RouterGroup, db *sql.DB, hub *chat.Hub) {
	s := Service{
		DB:  db,
		Hub: hub,
	}
	rg.POST("/conversations/private", s.createOrGetPrivate)
	rg.POST("/conversations/group", s.createGroup)
	rg.POST("/conversations/:id/participants", s.addParticipant)
	rg.DELETE("/conversations/:id/participants/:userId", s.removeParticipant)
	rg.GET("/conversations", s.listMine)
	rg.GET("/conversations/:id/participants", s.listParticipants)
}

func (s *Service) createOrGetPrivate(c *gin.Context) {
	uid := auth.MustUserID(c)
	var req privateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			fmt.Println("Validation errors:", validationErrors)
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		fmt.Println(c)
		fmt.Println("Binding error:", err)
		return
	}

	// find existing conversation
	row := s.DB.QueryRow(`SELECT c.id FROM conversations c
		JOIN participants p1 ON p1.conversation_id=c.id AND p1.user_id=$1
		JOIN participants p2 ON p2.conversation_id=c.id AND p2.user_id=$2
		WHERE c.is_group_chat=FALSE LIMIT 1`, uid, req.OtherUserId)

	var id int64
	if err := row.Scan(&id); err == nil {
		httpx.OK(c, gin.H{"conversation_id": id, "is_group": false})
		return
	}

	// start transaction
	tx, err := s.DB.Begin()
	if err != nil {
		httpx.Err(c, 500, "db transaction failed")
		return
	}
	defer tx.Rollback() // ensures cleanup on error

	// create conversation
	var conversationID int64
	err = tx.QueryRow(`INSERT INTO conversations (name, is_group_chat) VALUES (NULL, FALSE) RETURNING id`).Scan(&conversationID)
	if err != nil {
		httpx.Err(c, 400, "create conversation failed")
		fmt.Println("Insert conversation error:", err)
		return
	}

	// add participants (this will fail if user doesn't exist because of FK)
	_, err = tx.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES ($1, $2, FALSE), ($3, $4, FALSE)`,
		conversationID, uid, conversationID, req.OtherUserId)
	if err != nil {
		fmt.Println("Insert participants error:", err)
		httpx.Err(c, 400, "invalid user id")
		return
	}

	// commit if everything is fine
	if err := tx.Commit(); err != nil {
		httpx.Err(c, 500, "commit failed")
		return
	}
	s.Hub.BroadcastConversationUpdate(conversationID, "new_conversation") // Notify participants of new conversation

	httpx.OK(c, gin.H{"success": true, "conversation_id": conversationID, "is_group": false})
}

func (s *Service) createGroup(c *gin.Context) {
	uid := auth.MustUserID(c)
	var req groupReq

	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	// start transaction
	tx, err := s.DB.Begin()
	if err != nil {
		httpx.Err(c, 500, "db transaction failed")
		return
	}
	defer tx.Rollback()

	// create conversation
	var cid int64
	err = tx.QueryRow(`INSERT INTO conversations (name, is_group_chat) VALUES ($1, TRUE) RETURNING id`, req.Name).Scan(&cid)
	if err != nil {
		httpx.Err(c, 400, "create group failed")
		return
	}

	// insert creator as admin
	_, err = tx.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES ($1, $2, TRUE)`, cid, uid)
	if err != nil {
		httpx.Err(c, 400, "add creator failed")
		return
	}

	validMembers := 0

	// add other members if they exist
	for _, mid := range req.MemberIDs {
		if mid == uid {
			continue
		}

		var exists bool
		err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)", mid).Scan(&exists)
		if err != nil {
			httpx.Err(c, 500, "db error")
			return
		}
		if !exists {
			continue // skip invalid user
		}

		_, err = tx.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES ($1, $2, FALSE) ON CONFLICT DO NOTHING`, cid, mid)
		if err != nil {
			httpx.Err(c, 400, "add member failed")
			return
		}
		validMembers++
	}

	// if no valid members (except creator), rollback
	if validMembers == 0 {
		httpx.Err(c, 400, "no valid members found, group not created")
		return
	}

	// commit if everything is fine
	if err := tx.Commit(); err != nil {
		httpx.Err(c, 500, "commit failed")
		return
	}
	var username string
	_ = s.DB.QueryRow(`SELECT username FROM users WHERE id=$1`, uid).Scan(&username)

	s.Hub.BroadcastSystemMessage(cid, fmt.Sprintf("Group '%s' created by %s", req.Name, username))

	s.Hub.BroadcastConversationUpdate(cid, "new_conversation") // Notify participants of new conversation

	httpx.OK(c, gin.H{"success": true, "conversation_id": cid, "is_group": true})
}

func (s Service) addParticipant(c *gin.Context) {
	uid := auth.MustUserID(c)
	cid := c.Param("id")
	ncid, _ := strconv.ParseInt(cid, 10, 64) // Convert cid to int64

	//ensure uid is admin
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM participants WHERE conversation_id=$1 AND user_id=$2 AND is_admin=TRUE`, cid, uid).Scan(&n)
	if n == 0 {
		httpx.Err(c, http.StatusForbidden, "only admin can add participants")
		return
	}

	var req addReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	_, err := s.DB.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES ($1, $2, FALSE) ON CONFLICT DO NOTHING`, cid, req.UserID)
	var removedUsername string
	_ = s.DB.QueryRow(`SELECT username FROM users WHERE id=$1`, req.UserID).Scan(&removedUsername)
	if err != nil {
		httpx.Err(c, 400, "add failed")
		return
	}
	s.Hub.BroadcastSystemMessage(ncid, fmt.Sprintf("%s has been added to the group.", removedUsername))
	s.Hub.BroadcastConversationUpdate(ncid, "added_to_conversation") // Notify the added user
	httpx.OK(c, gin.H{"success": true})
}

func (s Service) removeParticipant(c *gin.Context) {
	uid := auth.MustUserID(c)
	cid := c.Param("id")
	ncid, _ := strconv.ParseInt(cid, 10, 64)

	var isAdmin bool
	_ = s.DB.QueryRow(`SELECT is_admin FROM participants WHERE conversation_id=$1 AND user_id=$2`, cid, uid).Scan(&isAdmin)
	if !isAdmin {
		httpx.Err(c, http.StatusForbidden, "only admin can remove participants")
		return
	}

	removedUserId, _ := strconv.ParseInt(c.Param("userId"), 10, 64)
	if uid == removedUserId {
		httpx.Err(c, http.StatusForbidden, "cannot remove yourself")
		return
	}

	// Get the removed user's username
	var removedUsername string
	_ = s.DB.QueryRow(`SELECT username FROM users WHERE id=$1`, removedUserId).Scan(&removedUsername)

	// Delete the participant
	_, err := s.DB.Exec(`DELETE FROM participants WHERE conversation_id=$1 AND user_id=$2`, cid, removedUserId)
	if err != nil {
		httpx.Err(c, 400, "remove failed")
		return
	}

	// Send the system message to the chat
	s.Hub.BroadcastSystemMessage(ncid, fmt.Sprintf("%s has been removed from the group.", removedUsername))

	// Notify the removed user
	s.Hub.BroadcastConversationUpdate(ncid, "removed_from_conversation")

	httpx.OK(c, gin.H{"success": true})
}

func (s Service) listMine(c *gin.Context) {
	uid := auth.MustUserID(c)

	rows, err := s.DB.Query(`
		SELECT
			c.id,
			c.name,
			c.is_group_chat,
			c.created_at,
			CASE WHEN c.is_group_chat = FALSE THEN other_user.username ELSE c.name END as display_name,
			CASE WHEN c.is_group_chat = FALSE THEN other_user.profile_pic ELSE NULL END as avatar,
			CASE WHEN c.is_group_chat = FALSE THEN other_user.last_active ELSE NULL END as last_active,
			CASE WHEN c.is_group_chat = FALSE THEN other_user.id ELSE NULL END as other_user_id,
			(SELECT COUNT(1) FROM participants WHERE conversation_id = c.id) AS participant_count,
			(SELECT m.content FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1) AS last_message,
			(SELECT m.sent_at FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1) AS last_message_at,
			COALESCE((
				SELECT COUNT(m.id)
				FROM messages m
				LEFT JOIN message_status ms ON m.id = ms.message_id AND ms.user_id = $1
				WHERE m.conversation_id = c.id
				AND m.sender_id != $2
				AND ms.status IS DISTINCT FROM 'read'   --just changed here
			), 0) AS unread_count
		FROM conversations c
		JOIN participants p1 ON p1.conversation_id = c.id
		LEFT JOIN participants p2 ON c.is_group_chat = FALSE AND p2.conversation_id = c.id AND p2.user_id != p1.user_id
		LEFT JOIN users other_user ON p2.user_id = other_user.id
		WHERE p1.user_id = $3
		GROUP BY c.id, c.name, c.is_group_chat, c.created_at, display_name, avatar, last_active, other_user_id
		ORDER BY last_message_at DESC NULLS LAST, c.created_at DESC`, uid, uid, uid)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "failed to fetch conversations")
		return
	}
	defer rows.Close()

	var list []gin.H

	for rows.Next() {
		var (
			id               int64
			name             sql.NullString
			isg              bool
			ca               time.Time // Use time.Time directly for PostgreSQL TIMESTAMP WITH TIME ZONE
			displayName      sql.NullString
			avatar           sql.NullString
			lastActive       sql.NullTime
			otherUserId      sql.NullInt64
			participantCount int64
			lastMessage      sql.NullString
			lastMessageAt    sql.NullTime
			unreadCount      int64
		)

		if err := rows.Scan(&id, &name, &isg, &ca, &displayName, &avatar, &lastActive, &otherUserId, &participantCount, &lastMessage, &lastMessageAt, &unreadCount); err != nil {
			fmt.Printf("listMine: failed to scan row: %v\n", err)
			continue
		}

		// Online check
		isOnline := false
		if lastActive.Valid {
			if time.Since(lastActive.Time) < time.Minute {
				isOnline = true
			}
		}

		// Base conversation object
		conversation := gin.H{
			"id":                id,
			"name":              displayName.String,
			"is_group":          isg,
			"participant_count": participantCount,
			"unread_count":      unreadCount,
			"avatar":            avatar.String,
			"is_online":         isOnline,
		}

		if otherUserId.Valid {
			conversation["other_user_id"] = otherUserId.Int64
		}

		// Add last_active timestamp
		if lastActive.Valid {
			conversation["last_seen"] = lastActive.Time.UTC().Format(time.RFC3339)
		}

		// Created_at
		conversation["created_at"] = ca.UTC().Format(time.RFC3339)

		// Last message (safe parse)
		if lastMessage.Valid && lastMessageAt.Valid {
			conversation["last_message"] = gin.H{
				"content":    lastMessage.String,
				"created_at": lastMessageAt.Time.UTC().Format(time.RFC3339),
			}
		}

		list = append(list, conversation)
	}

	if err := rows.Err(); err != nil {
		httpx.Err(c, http.StatusInternalServerError, "error reading conversation list")
		return
	}

	httpx.OK(c, gin.H{"success": true, "conversations": list})
}

// New function to list participants of a conversation
func (s *Service) listParticipants(c *gin.Context) {
	// Ensure the current user is a participant
	uid := auth.MustUserID(c)
	cid := c.Param("id")
	var isParticipant bool
	_ = s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM participants WHERE conversation_id=$1 AND user_id=$2)`, cid, uid).Scan(&isParticipant)
	if !isParticipant {
		httpx.Err(c, http.StatusForbidden, "not a member of this group")
		return
	}

	rows, err := s.DB.Query(`
		SELECT u.id, u.username, u.profile_pic, p.is_admin
		FROM participants p
		JOIN users u ON p.user_id = u.id
		WHERE p.conversation_id=$1`, cid)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var participants []gin.H
	for rows.Next() {
		var id int64
		var username string
		var profilePic sql.NullString
		var isAdmin bool
		if err := rows.Scan(&id, &username, &profilePic, &isAdmin); err != nil {
			continue
		}
		participants = append(participants, gin.H{
			"id":              id,
			"username":        username,
			"profile_picture": profilePic.String,
			"is_admin":        isAdmin,
		})
	}

	httpx.OK(c, gin.H{"success": true, "participants": participants})
}
