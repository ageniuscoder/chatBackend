package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ctxKey string //these two lines  ensures safe storage/retrieval in context.Context.
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
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Token"})
		}

		c.Set(string(CtxUserID), claims.UserId)
		c.Next()
	}
}

func MustUserId(c *gin.Context) int64 {
	if v, ok := c.Get(string(CtxUserID)); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}
