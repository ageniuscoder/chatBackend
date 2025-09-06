package profile

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ageniuscoder/mmchat/backend/internal/auth"
	"github.com/ageniuscoder/mmchat/backend/internal/httpx"
	"github.com/gin-gonic/gin"
)

type Service struct {
	DB *sql.DB
}
type UpdateReq struct {
	Username       string `json:"username"`
	ProfilePicture string `json:"profile_picture"`
}

func Register(rg *gin.RouterGroup, db *sql.DB) {
	s := Service{
		DB: db,
	}
	rg.GET("/me", s.getMe)
}

func (s Service) getMe(c *gin.Context) {
	uid := auth.MustUserID(c)

	if uid == 0 {
		httpx.Err(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	row := s.DB.QueryRow( //bug here at profile pic
		`SELECT id, username, phone_number, COALESCE(profile_pic, '') AS profile_pic, created_at 
		FROM users WHERE id=?`, uid,
	)

	var id int64
	var username, phone, pic string
	var created time.Time

	if err := row.Scan(&id, &username, &phone, &pic, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Err(c, http.StatusNotFound, "user not found")
		} else {
			fmt.Printf("[getMe] DB error: %v\n", err)
			httpx.Err(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	httpx.OK(c, gin.H{
		"id":              id,
		"username":        username,
		"phone_number":    phone,
		"profile_picture": pic,
		"created_at":      created,
	})
}
