package auth

import (
	"fmt"
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
		fmt.Println(claims)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Token"})
			return
		}

		c.Set(string(CtxUserID), int64(claims.UserId))
		c.Next()
	}
}

func MustUserID(c *gin.Context) int64 {
	v, ok := c.Get(string(CtxUserID))
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		panic("user_id missing from context")
	}
	id, ok := v.(int64)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		panic(fmt.Sprintf("user_id wrong type: %T", v))
	}
	return id
}
