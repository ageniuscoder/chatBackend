package config

import (
	"os"
	"strconv"
)

type Config struct {
	Addr      string
	JWTSecret string
	JWTTTLMin int
	SQLITEDsn string
	OTPDigits int
	OTPTTLSec int
	// ✅ SendGrid config
	SendGridAPIKey string
	SendGridFrom   string
}

func getenv(key, def string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return def
}

func MustLoad() Config {
	jwtttl, _ := strconv.Atoi(getenv("JWT_TTL_MIN", "1440"))
	otpdigit, _ := strconv.Atoi(getenv("OTP_DIGITS", "6"))
	otpttl, _ := strconv.Atoi(getenv("OTP_TTL_SEC", "300"))

	cfg := Config{
		Addr:           getenv("HTTP_ADDR", ":8080"),
		JWTSecret:      getenv("JWT_SECRET", ""),
		JWTTTLMin:      jwtttl,
		SQLITEDsn:      getenv("SQLITE_DSN", "file:chat.db?_pragma=foreign_keys(ON)"),
		OTPDigits:      otpdigit,
		OTPTTLSec:      otpttl,
		SendGridAPIKey: getenv("SENDGRID_API_KEY", ""),
		SendGridFrom:   getenv("SENDGRID_FROM", ""),
	}
	return cfg
}
