package messaging

import (
	"context"
	"fmt"
	"math/rand"
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

// humanDelay generates a random delay between min and max milliseconds
func humanDelay(minMs, maxMs int) time.Duration {
	if maxMs <= minMs {
		return time.Duration(minMs) * time.Millisecond
	}
	delay := minMs + rand.Intn(maxMs-minMs+1)
	return time.Duration(delay) * time.Millisecond
}

// randomSendDelay returns a random delay between 4 and 10 seconds
// instead of a fixed delay, to mimic human behavior
func randomSendDelay() time.Duration {
	return humanDelay(4000, 10000)
}

// simulateTyping simulates human typing behavior before sending a message.
// This sends online presence, typing indicator, waits proportionally to
// message length, then stops typing — mimicking natural human interaction.
func (s *Service) simulateTyping(client *whatsmeow.Client, recipient types.JID, messageLength int) {
	// 1. Set online presence so the recipient sees us as "online"
	if err := client.SendPresence(context.Background(), types.PresenceAvailable); err != nil {
		s.app.Logger.Printf("Warning: failed to send online presence: %v", err)
	}

	// 2. Pre-typing delay (humans don't start typing immediately)
	time.Sleep(humanDelay(500, 1500))

	// 3. Send "composing" (typing) indicator
	if err := client.SendChatPresence(context.Background(), recipient, types.ChatPresenceComposing, types.ChatPresenceMediaText); err != nil {
		s.app.Logger.Printf("Warning: failed to send composing presence: %v", err)
	}

	// 4. Typing duration proportional to message length
	//    Average human types ~40 chars/sec on phone, so ~25ms per char
	//    Minimum 1.5s, maximum 5s to avoid excessive waits
	typingMs := messageLength * 25
	if typingMs < 1500 {
		typingMs = 1500
	}
	if typingMs > 5000 {
		typingMs = 5000
	}
	// Add ±30% variation to make timing less predictable
	variation := typingMs * 30 / 100
	if variation > 0 {
		typingMs = typingMs - variation + rand.Intn(2*variation+1)
	}
	time.Sleep(time.Duration(typingMs) * time.Millisecond)

	// 5. Send "paused" (stopped typing) indicator
	if err := client.SendChatPresence(context.Background(), recipient, types.ChatPresencePaused, types.ChatPresenceMediaText); err != nil {
		s.app.Logger.Printf("Warning: failed to send paused presence: %v", err)
	}

	// 6. Small natural pause before the message actually sends
	time.Sleep(humanDelay(200, 500))
}

// SendMessage sends a text message to a WhatsApp contact
func (s *Service) SendMessage(user, phoneNumber, message string) error {
	const duplicateWindow = 15 * time.Second
	const duplicateMax = 3
	const duplicateMessageWindow = 15 * time.Second
	const duplicateMessageMax = 1

	// Check if phoneNumber is empty or only whitespace
	if strings.TrimSpace(phoneNumber) == "" {
		s.app.Logger.Printf("Warning: phone number is empty for user %s", user)
		return fmt.Errorf("phone number is empty, cannot send message")
	}
	// Check if phoneNumber is valid: all digits or starts with '+' followed by digits
	valid := true
	if phoneNumber[0] == '+' {
		if len(phoneNumber) == 1 {
			valid = false
		} else {
			for _, c := range phoneNumber[1:] {
				if c < '0' || c > '9' {
					valid = false
					break
				}
			}
		}
	} else {
		for _, c := range phoneNumber {
			if c < '0' || c > '9' {
				valid = false
				break
			}
		}
	}
	if !valid {
		s.app.Logger.Printf("Warning: phone number is invalid for user %s: %s", user, phoneNumber)
		return fmt.Errorf("phone number is invalid, must be all digits or start with '+' followed by digits")
	}

	dupKey := fmt.Sprintf("num|%s|%s", user, phoneNumber)
	allowed, retryAfter := s.app.DuplicateLimiter.Allow(dupKey, duplicateMax, duplicateWindow)
	if !allowed {
		return &DuplicateMessageError{RetryAfter: retryAfter}
	}

	msgKey := fmt.Sprintf("msg|%s|%s|%s", user, phoneNumber, message)
	msgAllowed, msgRetryAfter := s.app.DuplicateLimiter.Allow(msgKey, duplicateMessageMax, duplicateMessageWindow)
	if !msgAllowed {
		return &DuplicateMessageError{RetryAfter: msgRetryAfter}
	}

	// Use random delay instead of fixed delay to avoid bot detection
	s.app.SendLimiter.Wait(user, randomSendDelay())

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

		// === ANTI-BAN: Simulate human typing behavior ===
		s.simulateTyping(sess.Client, recipient, len(message))

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

		// Post-send: set presence back to unavailable after a random delay
		go func() {
			time.Sleep(humanDelay(2000, 5000))
			_ = sess.Client.SendPresence(context.Background(), types.PresenceUnavailable)
		}()

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := sess.Client.MarkRead(ctx, typedMessageIDs, time.Now(), toJIDObj, fromJIDObj, types.ReceiptTypeRead)
	if err != nil {
		return fmt.Errorf("failed to mark as read: %v", err)
	}

	return nil
}
