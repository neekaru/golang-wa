package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Handlers contains HTTP handlers for authentication
type Handlers struct {
	app     *app.App
	service *Service
}

// NewHandlers creates a new authentication handlers instance
func NewHandlers(app *app.App) *Handlers {
	return &Handlers{
		app:     app,
		service: NewService(app),
	}
}

// QRImageHandler handles generating a QR code for WhatsApp Web authentication
func (h *Handlers) QRImageHandler(c *gin.Context) {
	user := c.Query("user")
	if user == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user"})
		return
	}

	qrCode, err := h.service.GenerateQRCode(user)
	if err != nil {
		// If the user is already logged in, return a specific message
		if err.Error() == "session is already logged in and connected" {
			sess, exists := h.service.sessionService.FindSessionByUser(user)
			if exists {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Session is already logged in and connected. No QR code needed.",
					"status": map[string]interface{}{
						"logged_in": sess.IsLoggedIn,
						"connected": sess.Client.IsConnected(),
						"user":      user,
					},
				})
				return
			}
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"qrcode": "data:image/png;base64," + qrCode})
}
