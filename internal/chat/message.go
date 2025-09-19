package chat

type WireMessage struct {
	Type           string `json:"type"` // "message", "read_receipt", "typing_start", "typing_stop", "presence","edited_message","deleted_message"
	ConversationID int64  `json:"conversation_id,omitempty"`
	MessageID      int64  `json:"message_id,omitempty"`
	SenderID       int64  `json:"sender_id"`
	SenderUsername string `json:"sender_username,omitempty"`
	Content        string `json:"content,omitempty"` // used for presence = "online"/"offline"
	SentAt         string `json:"sent_at,omitempty"`
	LastActive     string `json:"last_active,omitempty"` // used for presence
}
