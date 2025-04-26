package media

import (
	"net/http"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":       mediaType + " sent successfully",
		"file_name": fileName,
	})
}
