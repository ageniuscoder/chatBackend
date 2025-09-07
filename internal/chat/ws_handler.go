package chat

import (
	"net/http"

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

func RegisterWS(rg *gin.RouterGroup, hub *Hub, jwtSecret string) {
	rg.GET("/ws", func(c *gin.Context) {
		uid := auth.MustUserID(c)

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
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
