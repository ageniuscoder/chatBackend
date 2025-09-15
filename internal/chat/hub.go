package chat

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
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
			//mark user online
			h.DB.Exec(`UPDATE users SET last_active=CURRENT_TIMESTAMP WHERE id=?`, client.UserID)
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true

			// Broadcast online presence
			h.BroadcastPresence(client.UserID, "online")
		case client := <-h.unregister:
			if set, ok := h.clients[client.UserID]; ok {
				if _, ok := set[client]; ok {
					delete(set, client)
					close(client.Send)
					if len(set) == 0 {
						delete(h.clients, client.UserID)
						// Mark last_active and broadcast offline
						h.DB.Exec(`UPDATE users SET last_active=CURRENT_TIMESTAMP WHERE id=?`, client.UserID)
						h.BroadcastPresence(client.UserID, "offline")
					}
				}
			}
		}
	}
}

// BroadcastMessage sends a JSON payload to all participants of a conversation.
func (h *Hub) BroadcastMessage(conversationID, senderID, messageID int64, content string) {
	// Fetch all participants (single query)
	rows, err := h.DB.Query(`SELECT user_id FROM participants WHERE conversation_id=? AND user_id!=?`, conversationID, senderID)
	if err != nil {
		log.Printf("[hub] failed to fetch participants for conversation %d: %v", conversationID, err)
		return
	}
	defer rows.Close()

	// Fetch sender username
	var senderUsername string
	if err := h.DB.QueryRow(`SELECT username FROM users WHERE id=?`, senderID).Scan(&senderUsername); err != nil {
		log.Printf("[hub] failed to fetch sender username for %d: %v", senderID, err)
		senderUsername = "unknown"
	}

	// Fetch sent_at timestamp
	var sentAt time.Time
	if err := h.DB.QueryRow(`SELECT sent_at FROM messages WHERE id=?`, messageID).Scan(&sentAt); err != nil {
		log.Printf("[hub] failed to fetch sent_at for message %d: %v", messageID, err)
		// Fallback to current time if DB query fails.
		sentAt = time.Now()
	}

	// Prepare wire message payload
	wire := WireMessage{
		Type:           "message",
		ConversationID: conversationID,
		MessageID:      messageID,
		SenderID:       senderID,
		SenderUsername: senderUsername,
		Content:        content,
		SentAt:         sentAt.Format(time.RFC3339), // FIX: Format the time.Time object to RFC3339
	}
	payload, err := json.Marshal(wire)
	if err != nil {
		log.Printf("[hub] failed to marshal wire message: %v", err)
		return
	}

	// Iterate participants & broadcast
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			log.Printf("[hub] failed to scan participant user_id: %v", err)
			continue
		}
		// Add this line to log the recipient's ID
		log.Printf("[hub] Attempting to send message %d to recipient %d", messageID, uid)
		// Mark delivered for everyone except sender
		if uid != senderID {
			if _, err := h.DB.Exec(
				`INSERT OR IGNORE INTO message_status (message_id, user_id, status)
				 VALUES (?, ?, 'delivered')`, messageID, uid); err != nil {
				log.Printf("[hub] failed to insert message_status for user %d: %v", uid, err)
			}
		}

		// Send over WebSocket if connected
		if set, ok := h.clients[uid]; ok {
			for client := range set {
				select {
				case client.Send <- payload:
				default:
					// slow/broken client → drop
					close(client.Send)
					delete(set, client)
					log.Printf("[hub] dropped slow client for user %d", uid)
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("[hub] row iteration error: %v", err)
	}
}

// New helper: notify participants when someone reads a message
func (h *Hub) BroadcastReadReceipt(messageID, readerID int64) {
	var convID int64
	err := h.DB.QueryRow(`SELECT conversation_id FROM messages WHERE id=?`, messageID).Scan(&convID)
	if err != nil {
		// could be sql.ErrNoRows or real DB error
		log.Printf("BroadcastReadReceipt: failed to get conversation_id for message %d: %v", messageID, err)
		return
	}

	wire := WireMessage{
		Type:           "read_receipt",
		ConversationID: convID,
		MessageID:      messageID,
		SenderID:       readerID,
	}
	payload, err := json.Marshal(wire)
	if err != nil {
		log.Printf("BroadcastReadReceipt: failed to marshal JSON: %v", err)
		return
	}

	rows, err := h.DB.Query(`SELECT user_id FROM participants WHERE conversation_id=? AND user_id<>?`, convID, readerID)
	if err != nil {
		log.Printf("BroadcastReadReceipt: failed to query participants for convID %d: %v", convID, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			log.Printf("BroadcastReadReceipt: failed to scan participant uid: %v", err)
			continue
		}

		if set, ok := h.clients[uid]; ok {
			for cli := range set {
				cli.Send <- payload // sent successful
			}
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("BroadcastReadReceipt: rows iteration error: %v", err)
	}
}

func (h *Hub) BroadcastTyping(convID, userID int64, eventType string) {
	var username string
	_ = h.DB.QueryRow(`SELECT username FROM users WHERE id=?`, userID).Scan(&username)

	wire := WireMessage{
		Type:           eventType, // "typing_start" or "typing_stop"
		ConversationID: convID,
		SenderID:       userID,
		SenderUsername: username,
	}
	payload, _ := json.Marshal(wire)

	rows, _ := h.DB.Query(`SELECT user_id FROM participants WHERE conversation_id=? AND user_id<>?`, convID, userID)
	defer rows.Close()

	for rows.Next() {
		var uid int64
		_ = rows.Scan(&uid)
		if set, ok := h.clients[uid]; ok {
			for cli := range set {
				select {
				case cli.Send <- payload:
				default:
					close(cli.Send)
					delete(set, cli)
				}
			}
		}
	}
}

// update: BroadcastPresence now includes last_active timestamp
func (h *Hub) BroadcastPresence(userID int64, status string) {
	var username string
	var lastActive time.Time
	// Fetch username and last_active timestamp in a single query
	_ = h.DB.QueryRow(`SELECT username, last_active FROM users WHERE id=?`, userID).Scan(&username, &lastActive)

	wire := WireMessage{
		Type:           "presence",
		SenderID:       userID,
		SenderUsername: username,
		Content:        status,
		LastActive:     lastActive.Format(time.RFC3339), // Use the new field
	}
	payload, _ := json.Marshal(wire)

	//Find all conversations the user belongs to
	rows, _ := h.DB.Query(`
        SELECT DISTINCT p2.user_id
        FROM participants p1
        JOIN participants p2 ON p1.conversation_id = p2.conversation_id
        WHERE p1.user_id = ? AND p2.user_id <> ?`,
		userID, userID,
	)
	defer rows.Close()
	// Broadcast to all other participants
	for rows.Next() {
		var uid int64
		_ = rows.Scan(&uid)
		if set, ok := h.clients[uid]; ok {
			for cli := range set {
				select {
				case cli.Send <- payload:
				default:
					close(cli.Send)
					delete(set, cli)
				}
			}
		}
	}
}

func (h *Hub) BroadcastConversationUpdate(conversationID int64, updateType string) {
	wire := WireMessage{
		Type:           "conversation_update",
		ConversationID: conversationID,
		Content:        updateType, // e.g., "new_conversation", "participant_added", "participant_removed"
	}
	payload, _ := json.Marshal(wire)

	// Fetch all participants of the conversation
	rows, err := h.DB.Query(`SELECT user_id FROM participants WHERE conversation_id=?`, conversationID)
	if err != nil {
		log.Printf("[hub] failed to fetch participants for broadcast update: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			log.Printf("[hub] failed to scan participant user_id for broadcast update: %v", err)
			continue
		}
		if set, ok := h.clients[uid]; ok {
			for client := range set {
				select {
				case client.Send <- payload:
				default:
					close(client.Send)
					delete(set, client)
					log.Printf("[hub] dropped slow client for user %d during broadcast update", uid)
				}
			}
		}
	}
}
