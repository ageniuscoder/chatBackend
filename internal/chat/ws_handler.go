package chat

import (
	"log"
	"net/http"

	"github.com/ageniuscoder/mmchat/backend/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Use CheckOrigin to allow connections from your frontend URL
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:5173" || origin == "https://chatfrontend-mijn.onrender.com"
	},
}

// RegisterWS mounts GET /ws for authenticated clients.
// The Gin context is automatically checked by JWTMiddleware
func RegisterWS(rg *gin.RouterGroup, hub *Hub, jwtSecret string) {
	rg.GET("/ws", func(c *gin.Context) {
		uid := auth.MustUserID(c)

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to establish WebSocket connection"})
			return
		}

		client := &Client{
			Hub:    hub,
			Conn:   conn,
			Send:   make(chan []byte, 256),
			UserID: uid,
		}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	})
}
