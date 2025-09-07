package messages

import (
	"database/sql"
	"net/http"

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

func Register(rg *gin.RouterGroup, db *sql.DB, hub *chat.Hub) {
	s := Service{
		DB:  db,
		Hub: hub,
	}
	rg.POST("/messages", s.send)
	rg.GET("/conversations/:id/messages", s.list)
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

	rows, err := s.DB.Query(`SELECT id, sender_id, content, sent_at
		FROM messages WHERE conversation_id=? ORDER BY sent_at DESC LIMIT ? OFFSET ?`, cid, q.Limit, q.Offset)
	if err != nil {
		httpx.Err(c, 500, "db error")
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id, sid int64
		var content, at string
		_ = rows.Scan(&id, &sid, &content, &at)
		// get sender username for convenience
		var uname string
		_ = s.DB.QueryRow(`SELECT username FROM users WHERE id=?`, sid).Scan(&uname)
		list = append(list, gin.H{
			"id": id, "sender_id": sid, "sender_username": uname,
			"content": content, "sent_at": at,
		})
	}
	httpx.OK(c, gin.H{"messages": list})
}
