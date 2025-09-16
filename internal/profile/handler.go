package profile

import (
	"database/sql"
	"errors"
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
type UpdateReq struct {
	Username       string `json:"username"`
	ProfilePicture string `json:"profile_picture"`
}

func Register(rg *gin.RouterGroup, db *sql.DB) {
	s := Service{
		DB: db,
	}
	rg.GET("/me", s.getMe)
	rg.PUT("/me", s.updateMe)
}

func (s Service) getMe(c *gin.Context) {
	uid := auth.MustUserID(c)

	if uid == 0 {
		httpx.Err(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	row := s.DB.QueryRow(
		`SELECT id, username, email, COALESCE(profile_pic, '') AS profile_pic, created_at
		FROM users WHERE id=$1`, uid,
	)

	var id int64
	var username, email, pic string
	var created time.Time

	if err := row.Scan(&id, &username, &email, &pic, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Err(c, http.StatusNotFound, "user not found")
		} else {
			fmt.Printf("[getMe] DB error: %v\n", err)
			httpx.Err(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	httpx.OK(c, gin.H{
		"success":         true,
		"id":              id,
		"username":        username,
		"email":           email,
		"profile_picture": pic,
		"created_at":      created,
	})
}

func (s Service) updateMe(c *gin.Context) {
	uid := auth.MustUserID(c)
	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	_, err := s.DB.Exec(
		`UPDATE users SET username=$1, profile_pic=$2 WHERE id=$3`,
		req.Username, req.ProfilePicture, uid,
	)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Profile Update failed")
	}
	s.getMe(c)
}
