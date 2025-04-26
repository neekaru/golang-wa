package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/neekaru/whatsappgo-bot/internal/app"
	"github.com/neekaru/whatsappgo-bot/internal/session"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// Service handles messaging-related business logic
type Service struct {
	app            *app.App
	sessionService *session.Service
}

// NewService creates a new messaging service
func NewService(app *app.App) *Service {
	return &Service{
		app:            app,
		sessionService: session.NewService(app),
	}
}

// SendMessage sends a text message to a WhatsApp contact
func (s *Service) SendMessage(user, phoneNumber, message string) error {
	sess, exists := s.sessionService.FindSessionByUser(user)
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Ensure client is connected before sending
	if !sess.Client.IsConnected() {
		err := sess.Client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}
	}

	// Create recipient JID
	recipient := types.JID{
		User:   phoneNumber,
		Server: "s.whatsapp.net",
	}

	// Create message and send
	msg := &waE2E.Message{
		Conversation: proto.String(message),
	}

	opts := whatsmeow.SendRequestExtra{
		ID: whatsmeow.GenerateMessageID(),
	}

	_, err := sess.Client.SendMessage(context.Background(), recipient, msg, opts)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	// Log successful message send
	s.app.Logger.Printf("Message sent successfully to %s from user %s", recipient.String(), user)

	return nil
}

// MarkRead marks messages as read
func (s *Service) MarkRead(user string, messageIDs []string, fromJID, toJID string) error {
	sess, exists := s.sessionService.FindSessionByUser(user)
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Convert string message IDs to types.MessageID
	typedMessageIDs := make([]types.MessageID, len(messageIDs))
	for i, id := range messageIDs {
		typedMessageIDs[i] = types.MessageID(id)
	}

	fromJIDObj := types.JID{User: fromJID, Server: "s.whatsapp.net"}
	toJIDObj := types.JID{User: toJID, Server: "s.whatsapp.net"}

	err := sess.Client.MarkRead(typedMessageIDs, time.Now(), fromJIDObj, toJIDObj, types.ReceiptTypeRead)
	if err != nil {
		return fmt.Errorf("failed to mark as read: %v", err)
	}

	return nil
}
