package chat

type WireMessage struct {
	Type           string `json:"type"` // "message", "read_receipt", "typing_start", "typing_stop"
	ConversationID int64  `json:"conversation_id"`
	MessageID      int64  `json:"message_id,omitempty"`
	SenderID       int64  `json:"sender_id"`
	SenderUsername string `json:"sender_username,omitempty"`
	Content        string `json:"content,omitempty"`
	SentAt         string `json:"sent_at,omitempty"`
}
