package conversations

import (
	"database/sql"
	"fmt"
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

	httpx.OK(c, gin.H{"conversation_id": id, "is_group": false})
}

func (s Service) createGroup(c *gin.Context) {
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

	res, err := s.DB.Exec(`INSERT INTO conversations (name, is_group_chat) VALUES (?, 1)`, req.Name)
	if err != nil {
		httpx.Err(c, 400, "create group failed")
		return
	}

	cid, _ := res.LastInsertId()

	_, _ = s.DB.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 1)`, cid, uid)

	for _, mid := range req.MemberIDs {
		if mid == uid {
			continue
		}
		_, _ = s.DB.Exec(`INSERT OR IGNORE INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 0)`, cid, mid)
	}

	httpx.OK(c, gin.H{"conversation_id": cid, "is_group": true})
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
	httpx.OK(c, gin.H{"ok": true})
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
	httpx.OK(c, gin.H{"ok": true})
}

func (s Service) listMine(c *gin.Context) {
	uid := auth.MustUserID(c)

	// Query all conversations where this user is a participant
	rows, err := s.DB.Query(`
		SELECT c.id, c.name, c.is_group_chat, c.created_at
		FROM conversations c
		JOIN participants p ON p.conversation_id = c.id
		WHERE p.user_id = ?
		ORDER BY c.created_at DESC`, uid)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "failed to fetch conversations")
		return
	}
	defer rows.Close()

	var list []gin.H

	for rows.Next() {
		var (
			id   int64
			name sql.NullString // to handle NULL safely
			isg  bool
			ca   string
		)

		if err := rows.Scan(&id, &name, &isg, &ca); err != nil {
			// If a row fails to scan, log it and continue to next row
			fmt.Printf("listMine: failed to scan row: %v\n", err)
			continue
		}

		// Use empty string if name is NULL (private chat case)
		displayName := ""
		if name.Valid {
			displayName = name.String
		}

		list = append(list, gin.H{
			"id":         id,
			"name":       displayName,
			"is_group":   isg,
			"created_at": ca,
		})
	}

	// Check for iteration errors
	if err := rows.Err(); err != nil {
		httpx.Err(c, http.StatusInternalServerError, "error reading conversation list")
		return
	}

	httpx.OK(c, gin.H{"conversations": list})
}
