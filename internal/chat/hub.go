package chat

import (
	"database/sql"
	"encoding/json"
)

type Hub struct {
	DB *sql.DB

	register   chan *Client
	unregister chan *Client

	// userID -> set of client connections (handles multi-tab/or mutlti device)
	clients map[int64]map[*Client]bool
}

func NewHub(db *sql.DB) *Hub {
	return &Hub{
		DB:         db,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[int64]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
		case client := <-h.unregister:
			if set, ok := h.clients[client.UserID]; ok {
				if _, ok := set[client]; ok {
					delete(set, client)
					close(client.Send)
					if len(set) == 0 {
						delete(h.clients, client.UserID)
					}
				}
			}
		}
	}
}

// BroadcastMessage sends a JSON payload to all participants of a conversation.
func (h *Hub) BroadcastMessage(conversationID, senderID, messageID int64, content string) {
	//Fetch Participants UserIds
	rows, err := h.DB.Query(`SELECT user_id FROM participants WHERE conversation_id=?`, conversationID)
	if err != nil {
		return
	}
	defer rows.Close()

	// Fetch sender username and timestamp
	var senderUsername string
	_ = h.DB.QueryRow(`SELECT username FROM users WHERE id=?`, senderID).Scan(&senderUsername)
	var sentAt string
	_ = h.DB.QueryRow(`SELECT sent_at FROM messages WHERE id=?`, messageID).Scan(&sentAt)

	wire := WireMessage{
		Type:           "message",
		ConversationID: conversationID,
		MessageID:      messageID,
		SenderID:       senderID,
		SenderUsername: senderUsername,
		Content:        content,
		SentAt:         sentAt,
	}

	payload, _ := json.Marshal(wire)

	for rows.Next() {
		var uid int64
		_ = rows.Scan(&uid)
		if set, ok := h.clients[uid]; ok {
			for client := range set {
				select {
				case client.Send <- payload:
				default:
					//slow/broken client, drop it
					close(client.Send)
					delete(set, client)
				}
			}
		}
	}

	// Mark delivered for others (optional best-effort)   //fetches all participants excepts sender
	rows2, err := h.DB.Query(`SELECT user_id FROM participants WHERE conversation_id=? AND user_id<>?`, conversationID, senderID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var uid int64
			_ = rows2.Scan(&uid)
			_, _ = h.DB.Exec(`INSERT OR IGNORE INTO message_status (message_id, user_id, status) VALUES (?, ?, 'delivered')`, messageID, uid)
		}
	}

}
