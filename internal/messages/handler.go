package messages

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time" // Import the time package for time formatting

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
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM participants WHERE conversation_id=? AND user_id=?`, req.ConversationID, uid).Scan(&n)
	if n == 0 {
		httpx.Err(c, http.StatusForbidden, "not a participant")
		return
	}

	res, err := s.DB.Exec(`INSERT INTO messages (conversation_id, sender_id, content) VALUES (?, ?, ?)`,
		req.ConversationID, uid, req.Content)
	if err != nil {
		httpx.Err(c, 400, "insert failed")
		return
	}
	mid, _ := res.LastInsertId()

	// fanout via hub (includes sender username in payload)
	s.Hub.BroadcastMessage(req.ConversationID, uid, mid, req.Content)

	httpx.OK(c, gin.H{"message_id": mid})
}

func (s Service) list(c *gin.Context) {
	cid := c.Param("id")
	var q pageReq
	_ = c.BindQuery(&q)
	if q.Limit <= 0 {
		q.Limit = 50
	}

	rows, err := s.DB.Query(`
		SELECT m.id, m.sender_id, u.username, m.content, m.sent_at
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.conversation_id=?
		ORDER BY m.sent_at DESC LIMIT ? OFFSET ?`, cid, q.Limit, q.Offset)
	if err != nil {
		httpx.Err(c, 500, "db error")
		return
	}
	defer rows.Close()

	var list []gin.H
	// In the list function, replace the `for rows.Next()` loop with this code:
	for rows.Next() {
		var id, sid int64
		var uname, content string
		var at sql.NullString // Use sql.NullString to handle potential NULL values

		if err := rows.Scan(&id, &sid, &uname, &content, &at); err != nil {
			fmt.Printf("list: failed to scan row: %v\n", err)
			continue
		}

		var sentAt string
		if at.Valid {
			parsedTime := utils.ParseTime(at.String)
			sentAt = parsedTime.Format(time.RFC3339)
		}

		list = append(list, gin.H{
			"id": id, "sender_id": sid, "sender_username": uname,
			"content": content, "sent_at": sentAt,
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

	// Build the IN clause for the SQL query
	if len(req.MessageIDs) == 0 {
		httpx.OK(c, gin.H{"message": "no messages to mark as read"})
		return
	}

	// Prepare the query with a dynamic number of placeholders
	placeholders := make([]string, len(req.MessageIDs))
	args := make([]interface{}, len(req.MessageIDs)+1)
	for i := range req.MessageIDs {
		placeholders[i] = "?"
		args[i] = req.MessageIDs[i]
	}
	args[len(req.MessageIDs)] = uid

	query := fmt.Sprintf(`INSERT INTO message_status (message_id, user_id, status, read_at)
		VALUES %s
		ON CONFLICT(message_id, user_id) DO UPDATE SET status='read', read_at=CURRENT_TIMESTAMP`,
		strings.TrimSuffix(strings.Repeat("(?, ?, 'read', CURRENT_TIMESTAMP),", len(req.MessageIDs)), ","))

	// Update message_status
	_, err := s.DB.Exec(query, args...)
	if err != nil {
		httpx.Err(c, http.StatusBadRequest, "db error")
		return
	}

	// Notify other participants via hub
	for _, mid := range req.MessageIDs {
		s.Hub.BroadcastReadReceipt(mid, uid)
	}

	httpx.OK(c, gin.H{"message": "marked as read"})
}
