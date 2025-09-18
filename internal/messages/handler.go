package messages

import (
	"database/sql"
	"fmt"
	"net/http"
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

type sendReq struct {
	ConversationID int64  `json:"conversation_id"`
	Content        string `json:"content"`
}

type pageReq struct {
	Limit  int `form:"limit"`
	Offset int `form:"offset"`
}

type readReq struct {
	MessageIDs []int64 `json:"message_ids"`
}

func Register(rg *gin.RouterGroup, db *sql.DB, hub *chat.Hub) {
	s := Service{
		DB:  db,
		Hub: hub,
	}
	rg.POST("/messages", s.send)
	rg.GET("/conversations/:id/messages", s.list)
	rg.POST("/messages/read", s.markRead)
}

func (s Service) send(c *gin.Context) {
	uid := auth.MustUserID(c)
	var req sendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	// authorize participant
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM participants WHERE conversation_id=$1 AND user_id=$2`, req.ConversationID, uid).Scan(&n)
	if n == 0 {
		httpx.Err(c, http.StatusForbidden, "not a participant")
		return
	}

	var mid int64
	err := s.DB.QueryRow(`INSERT INTO messages (conversation_id, sender_id, content) VALUES ($1, $2, $3) RETURNING id`,
		req.ConversationID, uid, req.Content).Scan(&mid)
	if err != nil {
		httpx.Err(c, 400, "insert failed")
		return
	}

	// fanout via hub (includes sender username in payload)
	s.Hub.BroadcastMessage(req.ConversationID, uid, mid, req.Content)

	httpx.OK(c, gin.H{"message_id": mid})
}

func (s Service) list(c *gin.Context) {
	uid := auth.MustUserID(c)
	cid := c.Param("id")
	var q pageReq
	_ = c.BindQuery(&q)
	if q.Limit <= 0 {
		q.Limit = 50
	}

	rows, err := s.DB.Query(`
		SELECT
			m.id,
			m.sender_id,
			u.username,
			m.content,
			m.sent_at,
			CASE
				WHEN m.sender_id = $1 THEN
					CASE WHEN EXISTS(
						SELECT 1 FROM participants p
						LEFT JOIN message_status ms ON ms.message_id = m.id AND ms.user_id = p.user_id
						WHERE p.conversation_id = m.conversation_id AND p.user_id != $1 AND ms.status != 'read'
					) THEN 'sent'
					ELSE 'read'
					END
				ELSE
					COALESCE(ms_receiver.status, 'delivered')
			END AS status
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		LEFT JOIN message_status ms_receiver ON ms_receiver.message_id = m.id AND ms_receiver.user_id = $1
		WHERE m.conversation_id = $2
		ORDER BY m.sent_at DESC
		LIMIT $3 OFFSET $4
	`, uid, cid, q.Limit, q.Offset)
	if err != nil {
		httpx.Err(c, 500, "db error")
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id, sid int64
		var uname, content, status string
		var at sql.NullTime

		if err := rows.Scan(&id, &sid, &uname, &content, &at, &status); err != nil {
			fmt.Printf("list: failed to scan row: %v\n", err)
			continue
		}

		var sentAt string
		if at.Valid {
			sentAt = at.Time.Format(time.RFC3339)
		}

		list = append(list, gin.H{
			"id": id, "sender_id": sid, "sender_username": uname,
			"content": content, "sent_at": sentAt, "status": status,
		})
	}
	httpx.OK(c, gin.H{"messages": list})
}

func (s Service) markRead(c *gin.Context) {
	uid := auth.MustUserID(c)
	var req readReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	if len(req.MessageIDs) == 0 {
		httpx.OK(c, gin.H{"message": "no messages to mark as read"})
		return
	}

	// Begin a transaction to ensure atomicity
	tx, err := s.DB.Begin()
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Use a loop to update each message individually for simplicity and robustness.
	for _, messageID := range req.MessageIDs {
		// First, check if the current user is a participant of the conversation
		// to which the message belongs.
		var conversationID int64
		err := tx.QueryRow(`SELECT conversation_id FROM messages WHERE id=$1`, messageID).Scan(&conversationID)
		if err != nil {
			fmt.Printf("Failed to get conversation_id for message %d: %v\n", messageID, err)
			continue
		}

		var isParticipant bool
		err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM participants WHERE conversation_id=$1 AND user_id=$2)`, conversationID, uid).Scan(&isParticipant)
		if err != nil || !isParticipant {
			fmt.Printf("User %d is not a participant of conversation %d\n", uid, conversationID)
			continue
		}

		// Update or Insert the message status.
		_, err = tx.Exec(`
			INSERT INTO message_status (message_id, user_id, status, read_at)
			VALUES ($1, $2, 'read', NOW())
			ON CONFLICT(message_id, user_id) DO UPDATE SET status='read', read_at=NOW()
		`, messageID, uid)
		if err != nil {
			fmt.Printf("Failed to mark message %d as read for user %d: %v\n", messageID, uid, err)
			continue
		}
		// Notify other participants via hub
		s.Hub.BroadcastReadReceipt(messageID, uid)
	}

	if err := tx.Commit(); err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	httpx.OK(c, gin.H{"message": "marked as read"})
}
