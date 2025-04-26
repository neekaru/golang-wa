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
	sessionCount := len(h.app.Sessions)
	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"uptime":        uptime,
		"session_count": sessionCount,
		"version":       "1.0.0",
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
		h.app.SessionsLock.RLock()
		defer h.app.SessionsLock.RUnlock()

		sessionCount = len(h.app.Sessions)
		for _, sess := range h.app.Sessions {
			if sess.IsLoggedIn {
				activeCount++
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
