package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/neekaru/whatsappgo-bot/internal/app"
	"github.com/neekaru/whatsappgo-bot/internal/session"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow/types/events"
)

// Service handles authentication-related business logic
type Service struct {
	app            *app.App
	sessionService *session.Service
}

// NewService creates a new authentication service
func NewService(app *app.App) *Service {
	return &Service{
		app:            app,
		sessionService: session.NewService(app),
	}
}

// GenerateQRCode generates a QR code for WhatsApp Web authentication
func (s *Service) GenerateQRCode(user string) (string, error) {
	sess, exists := s.sessionService.FindSessionByUser(user)
	if !exists {
		return "", fmt.Errorf("session not found")
	}

	// Check both logged_in and connection status
	if sess.IsLoggedIn && sess.Client.IsConnected() {
		s.app.Logger.Printf("User %s is already logged in and connected, no QR code needed", user)
		return "", fmt.Errorf("session is already logged in and connected")
	}

	// Always disconnect first to avoid "websocket is already connected" error
	// This is safe to call even if not connected
	if sess.Client.IsConnected() {
		s.app.Logger.Printf("Disconnecting existing connection for user %s before generating QR", user)
		sess.Client.Disconnect()
		// Small delay to ensure disconnection is complete
		time.Sleep(500 * time.Millisecond)
	}

	// Reset login state to be safe
	sess.IsLoggedIn = false

	// Set up a channel to receive the QR code
	qrCodeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start the client connection and QR code generation in a goroutine
	go func() {
		client := sess.Client

		// Set up event handlers before connecting
		qrChan, _ := client.GetQRChannel(context.Background())

		// Connect the client with error handling
		err := client.Connect()
		if err != nil {
			// Try to handle specific error types
			if strings.Contains(err.Error(), "websocket is already connected") {
				s.app.Logger.Printf("Got 'already connected' error for %s, trying to disconnect and reconnect", user)
				// Force disconnect and try again after a delay
				client.Disconnect()
				time.Sleep(1 * time.Second)
				err = client.Connect()
				if err != nil {
					errorChan <- fmt.Errorf("failed to connect client after retry: %v", err)
					return
				}
			} else {
				errorChan <- fmt.Errorf("failed to connect client: %v", err)
				return
			}
		}

		// Add connection event handler
		client.AddEventHandler(func(evt interface{}) {
			switch e := evt.(type) {
			case *events.Connected:
				sess.IsLoggedIn = true
				s.app.Logger.Printf("User %s connection state changed to: connected", user)
			case *events.LoggedOut:
				sess.IsLoggedIn = false
				// Inspect logout reason when available
				if e.OnConnect {
					s.app.Logger.Printf("User %s logged out on connect; reason=%s", user, e.Reason.String())
				} else {
					s.app.Logger.Printf("User %s logged out (stream error). Reason not provided in stream:error", user)
				}
			}
		})

		// Wait for QR code
		if qrChan != nil {
			select {
			case evt := <-qrChan:
				if evt.Code != "" {
					sess.QRLock.Lock()
					sess.LatestQRCode = evt.Code
					sess.QRLock.Unlock()

					s.app.Logger.Printf("Generated QR code for user %s", user)

					// Generate QR code image
					qr, err := qrcode.New(evt.Code, qrcode.Medium)
					if err != nil {
						errorChan <- fmt.Errorf("failed to generate QR code: %v", err)
						return
					}

					// Convert QR code to PNG bytes
					png, err := qr.PNG(256)
					if err != nil {
						errorChan <- fmt.Errorf("failed to generate PNG: %v", err)
						return
					}

					// Convert to base64
					qrBase64 := base64.StdEncoding.EncodeToString(png)
					qrCodeChan <- qrBase64
				} else {
					errorChan <- fmt.Errorf("received empty QR code")
				}
			case <-time.After(30 * time.Second):
				errorChan <- fmt.Errorf("timed out waiting for QR code generation")
			}
		} else {
			errorChan <- fmt.Errorf("failed to create QR channel")
		}
	}()

	// Wait for either the QR code or an error
	select {
	case qrCode := <-qrCodeChan:
		return qrCode, nil
	case err := <-errorChan:
		return "", err
	case <-time.After(60 * time.Second):
		return "", fmt.Errorf("QR code not available after waiting for 60 seconds")
	}
}
