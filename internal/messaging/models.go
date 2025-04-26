package messaging

// SendMessageRequest represents a request to send a text message
type SendMessageRequest struct {
	User        string `json:"user"`
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
}

// MarkReadRequest represents a request to mark messages as read
type MarkReadRequest struct {
	User      string   `json:"user"`
	MessageID []string `json:"message_ids"`
	FromJID   string   `json:"from_jid"`
	ToJID     string   `json:"to_jid"`
}
