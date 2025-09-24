package session

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Handlers contains HTTP handlers for session management
type Handlers struct {
	app     *app.App
	service *Service
}

// NewHandlers creates a new session handlers instance
func NewHandlers(app *app.App) *Handlers {
	return &Handlers{
		app:     app,
		service: NewService(app),
	}
}

// AddSessionHandler handles creating a new WhatsApp session
func (h *Handlers) AddSessionHandler(c *gin.Context) {
	var req AddSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	_, err := h.service.AddSession(req.User)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Inform the client that the session was created successfully, but QR generation is pending
	c.JSON(http.StatusOK, gin.H{"msg": "Session created. Please request QR code using /wa/qr-image"})
}

// StatusHandler handles checking the status of a WhatsApp session
func (h *Handlers) StatusHandler(c *gin.Context) {
	user := c.Query("user")
	if user == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user"})
		return
	}

	sess, exists := h.service.FindSessionByUser(user)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session not found"})
		return
	}

	// Get connection details
	isLoggedIn := sess.IsLoggedIn
	isConnected := sess.Client.IsConnected()

	// Log the status check
	h.app.Logger.Printf("Status check for user %s: logged_in=%v, connected=%v",
		user, isLoggedIn, isConnected)

	// Return detailed status
	c.JSON(http.StatusOK, gin.H{
		"logged_in": isLoggedIn,
		"connected": isConnected,
		"user":      user,
		"needs_qr":  !isLoggedIn || !isConnected,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// RestartHandler handles restarting a WhatsApp session
func (h *Handlers) RestartHandler(c *gin.Context) {
	user := c.Query("user")
	if user == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user"})
		return
	}

	h.app.Logger.Printf("Restarting session for user: %s", user)

	// First disconnect existing session if it exists
	if oldSess, exists := h.service.FindSessionByUser(user); exists {
		h.app.Logger.Printf("Disconnecting existing session for user: %s", user)
		// Safe disconnect with retry
		if oldSess.Client.IsConnected() {
			oldSess.Client.Disconnect()
			// Give it a moment to properly disconnect
			time.Sleep(500 * time.Millisecond)
		}

		// Remove from memory to force database restoration
		h.app.SessionsLock.Lock()
		delete(h.app.Sessions, user)
		h.app.SessionsLock.Unlock()
	}

	// Attempt to restore session from database
	sess, err := h.service.RestoreSession(user)
	if err != nil {
		h.app.Logger.Printf("Failed to restore session for user %s: %v", user, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore session: " + err.Error()})
		return
	}

	h.app.Logger.Printf("Session restored from database for user: %s", user)

	// Add restored session to memory
	h.app.SessionsLock.Lock()
	h.app.Sessions[user] = sess
	h.app.SessionsLock.Unlock()

	// Connect the restored session with retry logic
	err = sess.Client.Connect()
	if err != nil {
		// Try to handle specific error types
		if err.Error() == "websocket is already connected" {
			h.app.Logger.Printf("Got 'already connected' error for %s, trying to disconnect and reconnect", user)
			// Force disconnect and try again after a delay
			sess.Client.Disconnect()
			time.Sleep(1 * time.Second)
			err = sess.Client.Connect()
			if err != nil {
				h.app.Logger.Printf("Failed to connect after retry for user %s: %v", user, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to connect after retry: " + err.Error(),
					"status": map[string]any{
						"logged_in": sess.IsLoggedIn,
						"connected": sess.Client.IsConnected(),
						"user":      user,
					},
				})
				return
			}
		} else {
			h.app.Logger.Printf("Failed to connect for user %s: %v", user, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to connect restored session: " + err.Error(),
				"status": map[string]any{
					"logged_in": sess.IsLoggedIn,
					"connected": sess.Client.IsConnected(),
					"user":      user,
				},
			})
			return
		}
	}

	// Get connection details after reconnection
	isLoggedIn := sess.IsLoggedIn
	isConnected := sess.Client.IsConnected()

	h.app.Logger.Printf("Session successfully reconnected for user: %s (logged_in=%v, connected=%v)",
		user, isLoggedIn, isConnected)

	// Check if QR code is needed
	needsQR := !isLoggedIn || !isConnected

	// Prepare response message
	msg := "Session restored and connected successfully"
	if needsQR {
		h.app.Logger.Printf("Session needs QR code for user: %s", user)
		msg = "Session restored but needs QR code. Please request QR code using /wa/qr-image?user=" + user
	}

	c.JSON(http.StatusOK, gin.H{
		"msg": msg,
		"status": map[string]any{
			"logged_in": isLoggedIn,
			"connected": isConnected,
			"user":      user,
			"needs_qr":  needsQR,
		},
	})
}

// LogoutHandler handles logging out a WhatsApp session
func (h *Handlers) LogoutHandler(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Check if the session exists and get its connection status
	sess, exists := h.service.FindSessionByUser(req.User)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session not found"})
		return
	}

	isConnected := false
	if sess.Client != nil {
		isConnected = sess.Client.IsConnected()
	}

	// Provide appropriate feedback based on connection status
	if isConnected {
		c.JSON(http.StatusOK, gin.H{
			"msg": "Logout process started for connected session",
			"status": map[string]any{
				"user":      req.User,
				"connected": true,
				"logged_in": sess.IsLoggedIn,
			},
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"msg": "Session is disconnected. Will attempt to connect before logout.",
			"status": map[string]any{
				"user":      req.User,
				"connected": false,
				"logged_in": sess.IsLoggedIn,
			},
		})
	}

	// Process logout asynchronously to avoid blocking the response
	go func() {
		if err := h.service.LogoutSession(req.User); err != nil {
			h.app.Logger.Printf("Error during logout for %s: %v", req.User, err)
		}
	}()
}
