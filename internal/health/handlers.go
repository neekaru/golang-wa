package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Handlers contains HTTP handlers for health checks
type Handlers struct {
	app *app.App
}

// NewHandlers creates a new health handlers instance
func NewHandlers(app *app.App) *Handlers {
	return &Handlers{app: app}
}

// RootHandler handles the root endpoint for Docker health checks
func (h *Handlers) RootHandler(c *gin.Context) {
	uptime := time.Since(h.app.StartTime).String()

	// Use a try-lock approach to avoid deadlock
	var sessionCount int
	lockChan := make(chan struct{})

	go func() {
		// Count sessions from the legacy system
		h.app.SessionsLock.RLock()
		legacySessionMap := make(map[string]bool) // Track which users are in legacy sessions

		sessionCount = len(h.app.Sessions)
		for user := range h.app.Sessions {
			legacySessionMap[user] = true
		}
		h.app.SessionsLock.RUnlock()

		// Also count sessions from the ClientManager
		clientManager := h.app.GetClientManager()
		clients := clientManager.GetAllClients()
		for id := range clients {
			// Only count clients that aren't already counted in the legacy system
			if _, exists := legacySessionMap[id]; !exists {
				sessionCount++ // Increment total session count
			}
		}

		close(lockChan)
	}()

	// Wait for the lock with timeout
	select {
	case <-lockChan:
		// Lock acquired and data collected
	case <-time.After(500 * time.Millisecond):
		// Timeout - proceed with health check anyway
		h.app.Logger.Printf("Root health check timed out waiting for sessions lock")
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"uptime":        uptime,
		"session_count": sessionCount,
		"version":       "1.0.2",
	})
}

// HealthCheckHandler handles the health check endpoint
func (h *Handlers) HealthCheckHandler(c *gin.Context) {
	uptime := time.Since(h.app.StartTime).String()

	// Use a try-lock approach to avoid deadlock during session initialization
	// If we can't acquire the lock within a short time, proceed with partial data
	var sessionCount, activeCount int
	lockChan := make(chan struct{})

	go func() {
		// First count sessions from the legacy system
		h.app.SessionsLock.RLock()
		legacySessionMap := make(map[string]bool) // Track which users are in legacy sessions

		sessionCount = len(h.app.Sessions)
		for user, sess := range h.app.Sessions {
			legacySessionMap[user] = true
			if sess.IsLoggedIn {
				activeCount++
			}
		}
		h.app.SessionsLock.RUnlock()

		// Also count sessions from the ClientManager
		clientManager := h.app.GetClientManager()
		// Use exported methods to get clients
		clients := clientManager.GetAllClients()
		for id, client := range clients {
			// Only count clients that aren't already counted in the legacy system
			if _, exists := legacySessionMap[id]; !exists {
				sessionCount++ // Increment total session count
				if client.IsLoggedIn() {
					activeCount++
				}
			}
		}

		close(lockChan)
	}()

	// Wait for the lock with timeout
	select {
	case <-lockChan:
		// Lock acquired and data collected
	case <-time.After(500 * time.Millisecond):
		// Timeout - proceed with health check anyway
		h.app.Logger.Printf("Health check timed out waiting for sessions lock")
	}

	// Log health check access for debugging
	h.app.Logger.Printf("Health check requested from %s", c.ClientIP())

	// Always return 200 OK status
	c.JSON(http.StatusOK, gin.H{
		"status":          "ok",
		"uptime":          uptime,
		"total_sessions":  sessionCount,
		"active_sessions": activeCount,
		"timestamp":       time.Now().Format(time.RFC3339),
	})
}

// HealthCheckHandlerWithSlash handles the health check endpoint with trailing slash
func (h *Handlers) HealthCheckHandlerWithSlash(c *gin.Context) {
	h.HealthCheckHandler(c)
}
