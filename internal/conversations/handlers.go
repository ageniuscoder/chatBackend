package conversations

import (
	"database/sql"
	"net/http"

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

	// find existing
	row := s.DB.QueryRow(`SELECT c.id FROM conversations c
		JOIN participants p1 ON p1.conversation_id=c.id AND p1.user_id=?
		JOIN participants p2 ON p2.conversation_id=c.id AND p2.user_id=?
		WHERE c.is_group_chat=0 LIMIT 1`, uid, req.OtherUserId)

	var id int64
	if err := row.Scan(&id); err == nil {
		httpx.OK(c, gin.H{"conversation_id": id, "is_group": false})
		return
	}

	res, err := s.DB.Exec(`INSERT INTO conversations (name, is_group_chat) VALUES (NULL, 0)`)
	if err != nil {
		httpx.Err(c, 400, "create failed")
		return
	}
	id, _ = res.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 0), (?, ?, 0)`,
		id, uid, id, req.OtherUserId)
	httpx.OK(c, gin.H{"conversation_id": id, "is_group": false})

}
