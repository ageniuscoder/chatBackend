package profile

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	rg.GET("/users/:id/last-seen", s.getLastSeen)

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
		"success":         true,
		"id":              id,
		"username":        username,
		"phone_number":    phone,
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
		`UPDATE users SET username=?, profile_pic=? WHERE id=?`,
		req.Username, req.ProfilePicture, uid,
	)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Profile Update failed")
	}
	s.getMe(c)
}

func (s Service) getLastSeen(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		httpx.Err(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	row := s.DB.QueryRow(`SELECT last_active FROM users WHERE id=?`, userID)
	var lastActive time.Time
	if err := row.Scan(&lastActive); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Err(c, http.StatusNotFound, "user not found")
		} else {
			fmt.Printf("[getLastSeen] DB error: %v\n", err)
			httpx.Err(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	httpx.OK(c, gin.H{"success": true, "last_seen": lastActive.Format(time.RFC3339)})
}
