package users

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ageniuscoder/mmchat/backend/internal/auth"
	"github.com/ageniuscoder/mmchat/backend/internal/config"
	"github.com/ageniuscoder/mmchat/backend/internal/httpx"
	"github.com/ageniuscoder/mmchat/backend/internal/otp"
	"github.com/ageniuscoder/mmchat/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Service struct {
	DB        *sql.DB
	JWTSecret string
	JWTTTLMin int
	OTP       otp.Service
}

// ✅ Updated to use email
type signupInitReq struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type signupVerifyReq struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	OTP      string `json:"otp" binding:"required"`
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type forgotInitReq struct {
	Email string `json:"email" binding:"required,email"`
}

type forgotCompleteReq struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func RegisterPublic(rg *gin.RouterGroup, db *sql.DB, cfg config.Config) {
	s := Service{
		DB:        db,
		JWTSecret: cfg.JWTSecret,
		JWTTTLMin: cfg.JWTTTLMin,
		OTP: otp.Service{
			DB:             db,
			Digits:         cfg.OTPDigits,
			TTL:            time.Duration(cfg.OTPTTLSec) * time.Second,
			SendGridAPIKey: cfg.SendGridAPIKey,
			SendGridFrom:   cfg.SendGridFrom,
		},
	}

	rg.POST("/signup/initiate", s.signupInitiate)
	rg.POST("/signup/verify", s.signupVerify)
	rg.POST("/login", s.login)
	rg.POST("/logout", s.logout)
	rg.POST("/forgot/initiate", s.forgotInitiate)
	rg.POST("/forgot/reset", s.forgotComplete)
}

func (s Service) signupInitiate(c *gin.Context) {
	var req signupInitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	var count int
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM users WHERE username=? OR email=?`, req.Username, req.Email).Scan(&count)

	if count > 0 {
		httpx.Err(c, http.StatusConflict, "Username or Email Already Exists")
		return
	}

	if _, err := s.OTP.Genrate(req.Email, "signup"); err != nil {
		fmt.Println("otp generation error:", err)
		httpx.Err(c, http.StatusInternalServerError, "Otp Sent Failed")
		return
	}

	httpx.OK(c, gin.H{"success": true, "message": "Otp Sent"})
}

func (s Service) signupVerify(c *gin.Context) {
	var req signupVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	ok, err := s.OTP.Verify(req.Email, "signup", req.OTP)
	if err != nil || !ok {
		httpx.Err(c, 422, "Invalid Otp")
		return
	}
	hash, _ := auth.HashPassword(req.Password)
	res, err := s.DB.Exec(`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`, req.Username, req.Email, hash)
	if err != nil {
		httpx.Err(c, 400, "Create User Failed")
		return
	}

	uid, _ := res.LastInsertId()

	tok, err := auth.NewToken(s.JWTSecret, uid, s.JWTTTLMin)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Token Generation Failed")
		return
	}
	c.SetCookie("token", tok, s.JWTTTLMin*60, "/", "", true, true)

	httpx.OK(c, gin.H{"success": true, "user_id": uid})
}

func (s Service) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	row := s.DB.QueryRow(`SELECT id, password_hash FROM users WHERE username=?`, req.Username)

	var id int64
	var hash string
	if err := row.Scan(&id, &hash); err != nil {
		httpx.Err(c, http.StatusBadRequest, "Invalid Credentials")
		return
	}

	if err := auth.CheckPassword(hash, req.Password); err != nil {
		httpx.Err(c, http.StatusBadRequest, "Invalid Credentials")
		return
	}
	tok, _ := auth.NewToken(s.JWTSecret, id, s.JWTTTLMin)
	c.SetCookie("token", tok, s.JWTTTLMin*60, "/", "", true, true)
	httpx.OK(c, gin.H{"success": true, "user_id": id})
}

func (s Service) logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", true, true)
	httpx.OK(c, gin.H{"success": true, "message": "Logged out successfully"})
}

func (s Service) forgotInitiate(c *gin.Context) {
	var req forgotInitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := s.OTP.Genrate(req.Email, "reset"); err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Otp Sent Failed")
		return
	}
	httpx.OK(c, gin.H{"success": true, "message": "otp sent"})
}

func (s Service) forgotComplete(c *gin.Context) {
	var req forgotCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			httpx.Err(c, http.StatusBadRequest, utils.ValidationErr(validationErrors))
			fmt.Println("validation error:", utils.ValidationErr(validationErrors))
			return
		}
		httpx.Err(c, http.StatusBadRequest, err.Error())
		fmt.Println("bad request error:", err.Error())
		return
	}

	// Verify OTP and update password
	ok, err := s.OTP.Verify(req.Email, "reset", req.OTP)
	if err != nil || !ok {
		httpx.Err(c, http.StatusUnprocessableEntity, "Invalid Otp")
		return
	}

	hash, _ := auth.HashPassword(req.NewPassword)
	_, err = s.DB.Exec(`UPDATE users SET password_hash=? WHERE email=?`, hash, req.Email)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Update Password Failed")
		return
	}
	httpx.OK(c, gin.H{"success": true, "message": "password updated"})
}
