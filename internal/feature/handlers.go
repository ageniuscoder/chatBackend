package feature

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ageniuscoder/mmchat/backend/internal/httpx"
	"github.com/gin-gonic/gin"
)

type Service struct {
	DB *sql.DB
}

func Register(rg *gin.RouterGroup, db *sql.DB) {
	s := Service{
		DB: db,
	}
	rg.GET("/users/:id/last-seen", s.getLastSeen)
	rg.GET("/users/search", s.searchUsers)
}

func (s *Service) searchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		httpx.Err(c, http.StatusBadRequest, "query parameter is required")
		return
	}

	rows, err := s.DB.Query("SELECT id, username, profile_pic FROM users WHERE username LIKE $1 LIMIT 10", "%"+query+"%")
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "database query failed")
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var (
			id         int64
			username   string
			profilePic sql.NullString
		)
		if err := rows.Scan(&id, &username, &profilePic); err != nil {
			continue
		}
		users = append(users, gin.H{
			"id":              id,
			"username":        username,
			"profile_picture": profilePic.String,
		})
	}

	httpx.OK(c, gin.H{"success": true, "users": users})
}

func (s Service) getLastSeen(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		httpx.Err(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	row := s.DB.QueryRow(`SELECT last_active FROM users WHERE id=$1`, userID)
	var lastActive sql.NullTime
	if err := row.Scan(&lastActive); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Err(c, http.StatusNotFound, "user not found")
		} else {
			fmt.Printf("[getLastSeen] DB error: %v\n", err)
			httpx.Err(c, http.StatusInternalServerError, "database error")
		}
		return
	}

	httpx.OK(c, gin.H{"success": true, "last_seen": lastActive.Time.UTC().Format(time.RFC3339)})
}
