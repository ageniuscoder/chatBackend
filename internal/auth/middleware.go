package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ctxKey string

const CtxUserID ctxKey = "uid"

func JWTMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tok := strings.TrimPrefix(h, "Bearer ")
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
