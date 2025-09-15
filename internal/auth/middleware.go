package auth

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ctxKey string

const CtxUserID ctxKey = "uid"

func JWTMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok, err := c.Cookie("token")
		if err != nil {
			if err == http.ErrNoCookie {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token cookie"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "cookie read error"})
			return
		}

		claims, err := ParseToken(secret, tok)
		fmt.Println(claims)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Token"})
			return
		}

		c.Set(string(CtxUserID), int64(claims.UserId))
		c.Next()
	}
}

// added cors middleware
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	}
}

// UserIDFromContext retrieves the user ID from the context safely.
func UserIDFromContext(c *gin.Context) (int64, error) {
	v, ok := c.Get(string(CtxUserID))
	if !ok {
		return 0, fmt.Errorf("user_id missing from context")
	}
	id, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("user_id wrong type: %T", v)
	}
	return id, nil
}

// MustUserID is a convenience function that panics. Use UserIDFromContext for safer handling.
func MustUserID(c *gin.Context) int64 {
	id, err := UserIDFromContext(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		panic(err)
	}
	return id
}
