package media

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/neekaru/whatsappgo-bot/internal/app"
	"github.com/neekaru/whatsappgo-bot/internal/session"
	"github.com/neekaru/whatsappgo-bot/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Service handles media-related business logic
type Service struct {
	app            *app.App
	sessionService *session.Service
}

// NewService creates a new media service
func NewService(app *app.App) *Service {
	return &Service{
		app:            app,
		sessionService: session.NewService(app),
	}
}

// SendMedia sends media (image, video, file) to a WhatsApp contact
func (s *Service) SendMedia(user, phoneNumber, mediaType, mediaData, mediaURL, caption, fileName string) (string, error) {
	const sendDelay = 6 * time.Second

	// Check if phoneNumber is empty or only whitespace
	if strings.TrimSpace(phoneNumber) == "" {
		s.app.Logger.Printf("Warning: phone number is empty for user %s", user)
		return "", fmt.Errorf("phone number is empty, cannot send media")
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
		return "", fmt.Errorf("phone number is invalid, must be all digits or start with '+' followed by digits")
	}
	sess, exists := s.sessionService.FindSessionByUser(user)
	if !exists {
		return "", fmt.Errorf("session not found")
	}

	s.app.SendLimiter.Wait(user, sendDelay)

	// Ensure client is connected before sending
	if !sess.Client.IsConnected() {
		err := sess.Client.Connect()
		if err != nil {
			return "", fmt.Errorf("failed to connect: %v", err)
		}
	}

	recipient := types.JID{
		User:   phoneNumber,
		Server: "s.whatsapp.net",
	}

	var media []byte
	var mimeType string
	var detectedFileName string

	// Set filename if provided
	if fileName != "" {
		detectedFileName = fileName
	}

	// Check if URL or base64 media is provided
	if mediaURL != "" {
		// Download media from URL
		httpResp, err := http.Get(mediaURL)
		if err != nil {
			return "", fmt.Errorf("failed to download media from URL")
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download media")
		}

		media, err = io.ReadAll(httpResp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to download media")
		}

		mimeType = httpResp.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = http.DetectContentType(media)
		}

		// Extract filename from URL if not provided
		if detectedFileName == "" {
			// Parse URL to extract filename
			parsedURL, err := url.Parse(mediaURL)
			if err == nil {
				// Get the last part of the path
				parts := strings.Split(parsedURL.Path, "/")
				if len(parts) > 0 {
					urlFileName := parts[len(parts)-1]
					// Remove query parameters if present
					urlFileName = strings.Split(urlFileName, "?")[0]
					// Use it if it looks like a valid filename
					if urlFileName != "" && !strings.HasSuffix(urlFileName, "/") {
						detectedFileName = urlFileName
						s.app.Logger.Printf("Extracted filename from URL: %s", detectedFileName)
					}
				}
			}
		}

		// Try to get filename from Content-Disposition header if still not found
		if detectedFileName == "" {
			contentDisposition := httpResp.Header.Get("Content-Disposition")
			if contentDisposition != "" {
				if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
					if fn, ok := params["filename"]; ok && fn != "" {
						detectedFileName = fn
						s.app.Logger.Printf("Extracted filename from Content-Disposition: %s", detectedFileName)
					}
				}
			}
		}
	} else if mediaData != "" {
		// Decode base64 media
		var err error
		media, err = base64.StdEncoding.DecodeString(mediaData)
		if err != nil {
			return "", fmt.Errorf("invalid media format")
		}
		mimeType = http.DetectContentType(media)
	} else {
		return "", fmt.Errorf("either media or URL must be provided")
	}

	var waMediaType whatsmeow.MediaType
	switch mediaType {
	case "image":
		waMediaType = whatsmeow.MediaImage
	case "video":
		waMediaType = whatsmeow.MediaVideo
	case "file":
		waMediaType = whatsmeow.MediaDocument
	default:
		return "", fmt.Errorf("invalid media type: %s", mediaType)
	}

	uploaded, err := sess.Client.Upload(context.Background(), media, waMediaType)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %v", err)
	}

	var thumbnail []byte
	if mediaType == "video" {
		var errThumbnail error
		thumbnail, errThumbnail = utils.VideoThumbnail(
			media,
			0,
			struct{ Width int }{Width: 72},
		)

		if errThumbnail != nil {
			s.app.Logger.Printf("Failed to generate video thumbnail: %v", errThumbnail)
			thumbnail = nil // Proceed without a thumbnail if generation fails
		}
	}

	var msg waE2E.Message
	switch mediaType {
	case "image":
		msg = waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String(caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(media))),
			},
		}
	case "video":
		msg = waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:       proto.String(caption),
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				JPEGThumbnail: thumbnail,
			},
		}
	case "file":
		msg = waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				Caption:       proto.String(caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(media))),
				FileName:      proto.String(detectedFileName),
			},
		}
	}

	opts := whatsmeow.SendRequestExtra{
		ID: types.MessageID(fmt.Sprintf("%d", time.Now().UnixNano())),
	}

	// Use a context with a timeout for the SendMessage operation
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err = sess.Client.SendMessage(ctx, recipient, &msg, opts)
	if err != nil {
		// Check if this is a websocket disconnection error
		if strings.Contains(err.Error(), "websocket disconnected") {
			// Check if the user is logged in before attempting to reconnect
			if !sess.IsLoggedIn {
				s.app.Logger.Printf("User %s is not logged in, not attempting to reconnect", user)
				return "", fmt.Errorf("user is not logged in, cannot reconnect: %v", err)
			}

			s.app.Logger.Printf("Websocket disconnected during media send. Reconnecting...")

			// Disconnect explicitly to ensure clean state
			sess.Client.Disconnect()
			time.Sleep(1 * time.Second)

			// Try to reconnect
			err = sess.Client.Connect()
			if err != nil {
				s.app.Logger.Printf("Failed to reconnect: %v", err)
				return "", fmt.Errorf("failed to reconnect after websocket disconnection: %v", err)
			}

			s.app.Logger.Printf("Successfully reconnected, retrying media send")

			// Try sending again
			ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel2()

			_, err = sess.Client.SendMessage(ctx2, recipient, &msg, opts)
			if err != nil {
				return "", fmt.Errorf("failed to send media message after reconnection: %v", err)
			}
		} else {
			return "", fmt.Errorf("failed to send media message: %v", err)
		}
	}

	// Log successful message send
	s.app.Logger.Printf("Media sent successfully to %s from user %s", recipient.String(), user)

	return detectedFileName, nil
}
