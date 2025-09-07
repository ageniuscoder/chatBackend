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

func Register(rg *gin.RouterGroup, db *sql.DB, hub *chat.Hub) {
	s := Service{
		DB:  db,
		Hub: hub,
	}
	rg.POST("/messages", s.send)
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
