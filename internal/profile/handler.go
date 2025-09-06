package profile

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/ageniuscoder/mmchat/backend/internal/auth"
	"github.com/ageniuscoder/mmchat/backend/internal/httpx"
	"github.com/gin-gonic/gin"
)

type Service struct {
	DB *sql.DB
}
type getData struct {
	Id          int64     `json:"id"`
	Username    string    `json:"username"`
	PhoneNumber string    `json:"phone_number"`
	ProfilePic  string    `json:"profile_pic"`
	CreatedAt   time.Time `json:"created_at"`
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
	uid := auth.MustUserId(c)
	var data getData
	row := s.DB.QueryRow(`SELECT id, username, phone_number, profile_picture, created_at FROM users WHERE id=?`, uid)
	if err := row.Scan(&data); err != nil {
		httpx.Err(c, http.StatusNotFound, "User Not found")
		return
	}
	httpx.OK(c, gin.H{
		"user_id":     data.Id,
		"username":    data.Username,
		"phone":       data.PhoneNumber,
		"profile_pic": data.ProfilePic,
		"created_at":  data.CreatedAt,
	})

}
