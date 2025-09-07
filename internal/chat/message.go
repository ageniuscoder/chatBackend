package chat

type WireMessage struct {
	Type           string `json:"type"` // "message"
	ConversationID int64  `json:"conversation_id"`
	MessageID      int64  `json:"message_id"`
	SenderID       int64  `json:"sender_id"`
	SenderUsername string `json:"sender_username"`
	Content        string `json:"content"`
	SentAt         string `json:"sent_at,omitempty"`
}
