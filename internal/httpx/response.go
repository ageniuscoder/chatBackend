package httpx

import "github.com/gin-gonic/gin"

func OK(c *gin.Context, v any) {
	c.JSON(200, v)
}

func Err(c *gin.Context, code int, msg any) {
	c.JSON(code, gin.H{"error": msg})
}
