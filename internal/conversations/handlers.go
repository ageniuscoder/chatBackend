package conversations

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ageniuscoder/mmchat/backend/internal/auth"
	"github.com/ageniuscoder/mmchat/backend/internal/httpx"
	"github.com/ageniuscoder/mmchat/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Service struct {
	DB *sql.DB
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

func Register(rg *gin.RouterGroup, db *sql.DB) {
	s := Service{
		DB: db,
	}
	rg.POST("/conversations/private", s.createOrGetPrivate)
	rg.POST("/conversations/group", s.createGroup)
	rg.POST("/conversations/:id/participants", s.addParticipant)
	rg.DELETE("/conversations/:id/participants/:userId", s.removeParticipant)
	rg.GET("/conversations", s.listMine)
}

func (s *Service) createOrGetPrivate(c *gin.Context) {
	uid := auth.MustUserID(c)
	var req privateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	// find existing conversation
	row := s.DB.QueryRow(`SELECT c.id FROM conversations c
		JOIN participants p1 ON p1.conversation_id=c.id AND p1.user_id=?
		JOIN participants p2 ON p2.conversation_id=c.id AND p2.user_id=?
		WHERE c.is_group_chat=0 LIMIT 1`, uid, req.OtherUserId)

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
	res, err := tx.Exec(`INSERT INTO conversations (name, is_group_chat) VALUES (NULL, 0)`)
	if err != nil {
		httpx.Err(c, 400, "create conversation failed")
		return
	}
	id, _ = res.LastInsertId()

	// add participants (this will fail if user doesn't exist because of FK)
	_, err = tx.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 0), (?, ?, 0)`,
		id, uid, id, req.OtherUserId)
	if err != nil {
		httpx.Err(c, 400, "invalid user id")
		return
	}

	// commit if everything is fine
	if err := tx.Commit(); err != nil {
		httpx.Err(c, 500, "commit failed")
		return
	}

	httpx.OK(c, gin.H{"success": true, "conversation_id": id, "is_group": false})
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
	res, err := tx.Exec(`INSERT INTO conversations (name, is_group_chat) VALUES (?, 1)`, req.Name)
	if err != nil {
		httpx.Err(c, 400, "create group failed")
		return
	}
	cid, _ := res.LastInsertId()

	// insert creator as admin
	_, err = tx.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 1)`, cid, uid)
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
		err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id=?)", mid).Scan(&exists)
		if err != nil {
			httpx.Err(c, 500, "db error")
			return
		}
		if !exists {
			continue // skip invalid user
		}

		_, err = tx.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 0)`, cid, mid)
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

	httpx.OK(c, gin.H{"success": true, "conversation_id": cid, "is_group": true})
}

func (s Service) addParticipant(c *gin.Context) {
	uid := auth.MustUserID(c)
	cid := c.Param("id")

	//ensure uid is admin
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM participants WHERE conversation_id=? AND user_id=? AND is_admin=1`, cid, uid).Scan(&n)
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

	_, err := s.DB.Exec(`INSERT OR IGNORE INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 0)`, cid, req.UserID)
	if err != nil {
		httpx.Err(c, 400, "add failed")
		return
	}
	httpx.OK(c, gin.H{"success": true})
}

func (s Service) removeParticipant(c *gin.Context) {
	uid := auth.MustUserID(c)
	cid := c.Param("id")

	//ensure uid is admin
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM participants WHERE conversation_id=? AND user_id=? AND is_admin=1`, cid, uid).Scan(&n)
	if n == 0 {
		httpx.Err(c, http.StatusForbidden, "only admin can remove participants")
		return
	}
	//removing
	_, err := s.DB.Exec(`DELETE FROM participants WHERE conversation_id=? AND user_id=?`, cid, c.Param("userId"))
	if err != nil {
		httpx.Err(c, 400, "remove failed")
		return
	}
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
			CASE WHEN c.is_group_chat = 0 THEN other_user.username ELSE c.name END as display_name,
			CASE WHEN c.is_group_chat = 0 THEN other_user.profile_pic ELSE NULL END as avatar,
			CASE WHEN c.is_group_chat = 0 THEN other_user.last_active ELSE NULL END as last_active,
			(SELECT COUNT(1) FROM participants WHERE conversation_id = c.id) AS participant_count,
			(SELECT m.content FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1) AS last_message,
			(SELECT m.sent_at FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1) AS last_message_at,
			COALESCE((SELECT COUNT(1)
				FROM messages
				WHERE conversation_id = c.id
				AND id NOT IN (SELECT message_id FROM message_status WHERE user_id = ? AND status = 'read')), 0) AS unread_count
		FROM conversations c
		JOIN participants p1 ON p1.conversation_id = c.id
		LEFT JOIN participants p2 ON c.is_group_chat = 0 AND p2.conversation_id = c.id AND p2.user_id != p1.user_id
		LEFT JOIN users other_user ON p2.user_id = other_user.id
		WHERE p1.user_id = ?
		GROUP BY c.id
		ORDER BY last_message_at DESC, c.created_at DESC`, uid, uid)
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
			ca               string
			displayName      sql.NullString
			avatar           sql.NullString
			lastActive       sql.NullString
			participantCount int64
			lastMessage      sql.NullString
			lastMessageAt    sql.NullString
			unreadCount      int64
		)

		if err := rows.Scan(&id, &name, &isg, &ca, &displayName, &avatar, &lastActive, &participantCount, &lastMessage, &lastMessageAt, &unreadCount); err != nil {
			fmt.Printf("listMine: failed to scan row: %v\n", err)
			continue
		}

		// Online check
		isOnline := false
		if lastActive.Valid {
			t := utils.ParseTime(lastActive.String)
			if !t.IsZero() && time.Since(t) < time.Minute {
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

		// Created_at (safe parse)
		if t := utils.ParseTime(ca); !t.IsZero() {
			conversation["created_at"] = t.UTC().Format(time.RFC3339)
		}

		// Last message (safe parse)
		if lastMessage.Valid && lastMessageAt.Valid {
			if t := utils.ParseTime(lastMessageAt.String); !t.IsZero() {
				conversation["last_message"] = gin.H{
					"content":    lastMessage.String,
					"created_at": t.UTC().Format(time.RFC3339),
				}
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
