package messaging

import (
	"context"
	"fmt"
	"strings"
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
	return s.sendMessageWithRetry(user, phoneNumber, message)
}

// sendMessageWithRetry attempts to send a message with automatic reconnection and retry
// if a websocket disconnection error occurs
func (s *Service) sendMessageWithRetry(user, phoneNumber, message string) error {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Get the session
		sess, exists := s.sessionService.FindSessionByUser(user)
		if !exists {
			return fmt.Errorf("session not found")
		}

		// Ensure client is connected before sending
		if !sess.Client.IsConnected() {
			err := sess.Client.Connect()
			if err != nil {
				s.app.Logger.Printf("Failed to connect on attempt %d: %v", attempt+1, err)
				lastErr = fmt.Errorf("failed to connect: %v", err)
				time.Sleep(1 * time.Second)
				continue
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

		// Use a context with a longer timeout (60 seconds) for message sending operations
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

		// Send the message
		_, err := sess.Client.SendMessage(ctx, recipient, msg, opts)
		cancel() // Cancel the context after sending

		if err != nil {
			lastErr = fmt.Errorf("failed to send message: %v", err)

			// Check if this is a websocket disconnection error
			if strings.Contains(err.Error(), "websocket disconnected") {
				// Check if the user is logged in before attempting to reconnect
				if !sess.IsLoggedIn {
					s.app.Logger.Printf("User %s is not logged in, not attempting to reconnect", user)
					return fmt.Errorf("user is not logged in, cannot reconnect: %v", lastErr)
				}

				s.app.Logger.Printf("Websocket disconnected during message send (attempt %d/%d). Reconnecting...",
					attempt+1, maxRetries)

				// Disconnect explicitly to ensure clean state
				sess.Client.Disconnect()
				time.Sleep(1 * time.Second)

				// Try to reconnect
				err = sess.Client.Connect()
				if err != nil {
					s.app.Logger.Printf("Failed to reconnect on attempt %d: %v", attempt+1, err)
				} else {
					s.app.Logger.Printf("Successfully reconnected on attempt %d, retrying message send", attempt+1)
				}

				// Continue to next attempt
				continue
			}

			// For other types of errors, return immediately
			return lastErr
		}

		// If we get here, the message was sent successfully
		s.app.Logger.Printf("Message sent successfully to %s from user %s", recipient.String(), user)
		return nil
	}

	// If we've exhausted all retries, return the last error
	return lastErr
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

	// Use a context with a timeout for the MarkRead operation
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := sess.Client.MarkRead(typedMessageIDs, time.Now(), fromJIDObj, toJIDObj, types.ReceiptTypeRead)
	if err != nil {
		return fmt.Errorf("failed to mark as read: %v", err)
	}

	return nil
}
