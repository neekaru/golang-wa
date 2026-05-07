package app

import (
	"log"
	"sync"
	"time"

	"github.com/neekaru/whatsappgo-bot/internal/client"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// Session represents a WhatsApp session
// Kept for backward compatibility
type Session struct {
	Client       *whatsmeow.Client
	Container    *sqlstore.Container
	User         string
	Phone        string
	IsLoggedIn   bool
	LatestQRCode string       // Store the latest QR code
	QRLock       sync.RWMutex // Lock to protect access to LatestQRCode
}

// App holds shared application state and resources
type App struct {
	// Legacy session management - kept for backward compatibility
	Sessions     map[string]*Session
	SessionsLock sync.RWMutex

	Logger    *log.Logger
	StartTime time.Time // Track startup time for health checks

	SendLimiter *SendRateLimiter
	DuplicateLimiter *DuplicateMessageLimiter
}

// SendRateLimiter enforces a minimum delay between send operations per user.
type SendRateLimiter struct {
	mu          sync.Mutex
	nextAllowed map[string]time.Time
}

// NewSendRateLimiter creates a new SendRateLimiter.
func NewSendRateLimiter() *SendRateLimiter {
	return &SendRateLimiter{
		nextAllowed: make(map[string]time.Time),
	}
}

// Wait blocks until the caller is allowed to send for the given user.
func (l *SendRateLimiter) Wait(user string, delay time.Duration) {
	if delay <= 0 {
		return
	}

	now := time.Now()

	l.mu.Lock()
	next := l.nextAllowed[user]
	if next.After(now) {
		l.nextAllowed[user] = next.Add(delay)
		l.mu.Unlock()
		time.Sleep(time.Until(next))
		return
	}

	l.nextAllowed[user] = now.Add(delay)
	l.mu.Unlock()
}

// DuplicateMessageLimiter blocks repeated messages per key for a fixed window.
type DuplicateMessageLimiter struct {
	mu      sync.Mutex
	entries map[string]duplicateEntry
}

type duplicateEntry struct {
	windowStart time.Time
	count       int
}

// NewDuplicateMessageLimiter creates a new DuplicateMessageLimiter.
func NewDuplicateMessageLimiter() *DuplicateMessageLimiter {
	return &DuplicateMessageLimiter{
		entries: make(map[string]duplicateEntry),
	}
}

// Allow returns whether a message is allowed based on max sends in a window.
func (l *DuplicateMessageLimiter) Allow(key string, max int, window time.Duration) (bool, time.Duration) {
	if max <= 0 || window <= 0 {
		return true, 0
	}

	now := time.Now()

	l.mu.Lock()
	entry := l.entries[key]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) > window {
		entry.windowStart = now
		entry.count = 0
	}

	if entry.count >= max {
		retryAfter := window - now.Sub(entry.windowStart)
		if retryAfter < 0 {
			retryAfter = 0
		}
		l.mu.Unlock()
		return false, retryAfter
	}

	entry.count++
	l.entries[key] = entry
	l.mu.Unlock()

	return true, 0
}

// NewApp creates a new App instance with initialized resources
func NewApp(logger *log.Logger) *App {
	// Initialize the ClientManager singleton
	_ = client.GetInstance()

	return &App{
		Sessions:  make(map[string]*Session),
		Logger:    logger,
		StartTime: time.Now(),
		SendLimiter: NewSendRateLimiter(),
		DuplicateLimiter: NewDuplicateMessageLimiter(),
	}
}

// GetClientManager returns the ClientManager singleton
func (a *App) GetClientManager() *client.ClientManager {
	return client.GetInstance()
}
