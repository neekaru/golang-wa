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
}

// NewApp creates a new App instance with initialized resources
func NewApp(logger *log.Logger) *App {
	// Initialize the ClientManager singleton
	_ = client.GetInstance()

	return &App{
		Sessions:  make(map[string]*Session),
		Logger:    logger,
		StartTime: time.Now(),
	}
}

// GetClientManager returns the ClientManager singleton
func (a *App) GetClientManager() *client.ClientManager {
	return client.GetInstance()
}
