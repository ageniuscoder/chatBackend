package users

import (
	"database/sql"
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

type signupInitReq struct {
	Username string `json:"username" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type signupVerifyReq struct {
	Username string `json:"username" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"` // send again on verify
	OTP      string `json:"otp" binding:"required"`
}

type loginReq struct {
	Username string `json:"username" binding:"required" `
	Password string `json:"password" binding:"required"`
}

type forgotInitReq struct {
	Phone string `json:"phone" binding:"required"`
}

type forgotVerifyReq struct {
	Phone string `json:"phone" binding:"required"`
	OTP   string `json:"otp" binding:"required"`
}

type resetReq struct {
	Phone       string `json:"phone" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func RegisterPublic(rg *gin.RouterGroup, db *sql.DB, cfg config.Config) {
	s := Service{
		DB:        db,
		JWTSecret: cfg.JWTSecret,
		JWTTTLMin: cfg.JWTTTLMin,
		OTP: otp.Service{
			DB:     db,
			Digits: cfg.OTPDigits,
			TTL:    time.Duration(cfg.OTPTTLSec) * time.Second,
		},
	}

	rg.POST("/signup/initiate", s.signupInitiate)
	rg.POST("/signup/verify", s.signupVerify)
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
	_ = s.DB.QueryRow(`SELECT COUNT(1) FROM users WHERE username=? OR phone_number=?`, req.Username, req.Phone).Scan(&count)

	if count > 0 {
		httpx.Err(c, http.StatusConflict, "Username or Phone Already Exists")
		return
	}

	if _, err := s.OTP.Genrate(req.Phone, "signup"); err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Otp Sent Failed")
		return
	}

	httpx.OK(c, gin.H{"message": "Otp Sent"})
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

	ok, err := s.OTP.Verify(req.Phone, "signup", req.OTP)
	if err != nil || !ok {
		httpx.Err(c, http.StatusUnauthorized, "Invalid Otp")
		return
	}
	hash, _ := auth.HashPassword(req.Password)
	res, err := s.DB.Exec(`INSERT INTO users (username, phone_number, password_hash) VALUES (?, ?, ?)`, req.Username, req.Phone, hash)
	if err != nil {
		httpx.Err(c, 400, "Create User Failed")
		return
	}

	uid, _ := res.LastInsertId()

	tok, err := auth.NewToken(s.JWTSecret, uid, s.JWTTTLMin)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Token Genration Failed")
	}

	httpx.OK(c, gin.H{"token": tok, "user_id": uid})

}
