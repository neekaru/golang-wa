package media

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Handlers contains HTTP handlers for media
type Handlers struct {
	app     *app.App
	service *Service
}

// NewHandlers creates a new media handlers instance
func NewHandlers(app *app.App) *Handlers {
	return &Handlers{
		app:     app,
		service: NewService(app),
	}
}

// SendFileHandler handles sending a file
func (h *Handlers) SendFileHandler(c *gin.Context) {
	h.sendMediaHandler(c, "file")
}

// SendImageHandler handles sending an image
func (h *Handlers) SendImageHandler(c *gin.Context) {
	h.sendMediaHandler(c, "image")
}

// SendVideoHandler handles sending a video
func (h *Handlers) SendVideoHandler(c *gin.Context) {
	h.sendMediaHandler(c, "video")
}

// sendMediaHandler is a common handler for sending media
func (h *Handlers) sendMediaHandler(c *gin.Context, mediaType string) {
	var req SendMediaRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	fileName, err := h.service.SendMedia(
		req.User,
		req.PhoneNumber,
		mediaType,
		req.Media,
		req.URL,
		req.Caption,
		req.FileName,
	)
	if err != nil {
		// Log the detailed error
		h.app.Logger.Printf("Media send error for type %s: %v", mediaType, err)

		// Special handling for empty phone number
		if err.Error() == "phone number is empty, cannot send media" {
			c.JSON(http.StatusOK, gin.H{
				"error":   "Media cannot be send",
				"details": err.Error(),
			})
			return
		}
		// Special handling for invalid phone number format
		if err.Error() == "phone number is invalid, must contain only digits" {
			c.JSON(http.StatusOK, gin.H{
				"error":   "Media cannot be send",
				"details": err.Error(),
			})
			return
		}

		// Check if error is related to file/URL access
		if err.Error() == "failed to download media from URL" ||
			strings.Contains(err.Error(), "failed to download media") ||
			strings.Contains(err.Error(), "invalid media format") ||
			strings.Contains(err.Error(), "failed to upload media") ||
			strings.Contains(err.Error(), "failed to send media message") {
			c.JSON(http.StatusOK, gin.H{
				"msg":     "file/url cannot be send",
				"details": err.Error(),
			})
			return
		}

		// For other types of errors, still return 200 but with different message
		c.JSON(http.StatusOK, gin.H{
			"msg":     "file/url cannot be send",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":       mediaType + " sent successfully",
		"file_name": fileName,
	})
}
