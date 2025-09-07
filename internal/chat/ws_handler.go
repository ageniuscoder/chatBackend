package chat

import (
	"net/http"
	"strings"

	"github.com/ageniuscoder/mmchat/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow CORS for demo; tighten in prod.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// RegisterWS mounts GET /ws for authenticated clients.
// Auth works via:
// 1) Header: Authorization: Bearer <JWT>
// 2) Query:  ?token=<JWT>

func RegisterWS(rg *gin.RouterGroup, hub *Hub, jwtSecret string) {
	rg.GET("/ws", func(c *gin.Context) {
		// Extract token
		token := c.Query("token")
		if token == "" {
			h := c.GetHeader("Authorization")
			if strings.HasPrefix(h, "Bearer ") {
				token = strings.TrimPrefix(h, "Bearer ")
			}
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		cl, err := auth.ParseToken(jwtSecret, token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		client := &Client{
			Hub:    hub,
			Conn:   conn,
			Send:   make(chan []byte, 256),
			UserID: cl.UserId,
		}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	})
}
