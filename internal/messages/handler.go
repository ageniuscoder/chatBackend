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

	for _, messageID := range req.MessageIDs {
		var conversationID int64
		var senderID int64
		var isGroupChat bool

		// Change 1: Check if the current user is a participant and get conversation details.
		// We get `conversation_id`, `sender_id`, and `is_group_chat` in a single query.
		err := tx.QueryRow(`
			SELECT m.conversation_id, m.sender_id, c.is_group_chat
			FROM messages m
			JOIN conversations c ON c.id = m.conversation_id
			JOIN participants p ON p.conversation_id = m.conversation_id
			WHERE m.id = $1 AND p.user_id = $2
		`, messageID, uid).Scan(&conversationID, &senderID, &isGroupChat)

		if err != nil {
			fmt.Printf("Failed to validate message %d or user %d is not a participant: %v\n", messageID, uid, err)
			continue
		}

		// Update or Insert the message status for the current user.
		_, err = tx.Exec(`
			INSERT INTO message_status (message_id, user_id, status, read_at)
			VALUES ($1, $2, 'read', NOW())
			ON CONFLICT(message_id, user_id) DO UPDATE SET status='read', read_at=NOW()
		`, messageID, uid)
		if err != nil {
			fmt.Printf("Failed to mark message %d as read for user %d: %v\n", messageID, uid, err)
			continue
		}

		// Change 2: Check if this is a group chat.
		if isGroupChat {
			// Get the count of participants who have read this message
			var readCount int64
			err = tx.QueryRow(`SELECT COUNT(1) FROM message_status WHERE message_id = $1 AND status = 'read'`, messageID).Scan(&readCount)
			if err != nil {
				fmt.Printf("Failed to get read count for message %d: %v\n", messageID, err)
				continue
			}

			// Get the count of participants in the conversation, excluding the sender
			var totalParticipants int64
			err := tx.QueryRow(`SELECT COUNT(1) FROM participants WHERE conversation_id = $1 AND user_id != $2`, conversationID, senderID).Scan(&totalParticipants)
			if err != nil {
				fmt.Printf("Failed to get total participants for group %d: %v\n", conversationID, err)
				continue
			}

			// Change 3: Broadcast a read receipt ONLY if all other participants have read the message.
			if readCount == totalParticipants {
				s.Hub.BroadcastReadReceipt(messageID, uid)
			}
		} else { // For a private chat, always notify the sender.
			s.Hub.BroadcastReadReceipt(messageID, uid)
		}
	}

	if err := tx.Commit(); err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	httpx.OK(c, gin.H{"message": "marked as read"})
}
