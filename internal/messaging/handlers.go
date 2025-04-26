package messaging

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Handlers contains HTTP handlers for messaging
type Handlers struct {
	app     *app.App
	service *Service
}

// NewHandlers creates a new messaging handlers instance
func NewHandlers(app *app.App) *Handlers {
	return &Handlers{
		app:     app,
		service: NewService(app),
	}
}

// SendMessageHandler handles sending a text message
func (h *Handlers) SendMessageHandler(c *gin.Context) {
	var req SendMessageRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := h.service.SendMessage(req.User, req.PhoneNumber, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "Message sent successfully"})
}

// MarkReadHandler handles marking messages as read
func (h *Handlers) MarkReadHandler(c *gin.Context) {
	var req MarkReadRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := h.service.MarkRead(req.User, req.MessageID, req.FromJID, req.ToJID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "Messages marked as read"})
}
