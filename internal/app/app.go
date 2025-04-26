package app

import (
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// Session represents a WhatsApp session
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
	Sessions     map[string]*Session
	SessionsLock sync.RWMutex
	Logger       *log.Logger
	StartTime    time.Time // Track startup time for health checks
}

// NewApp creates a new App instance with initialized resources
func NewApp(logger *log.Logger) *App {
	return &App{
		Sessions:  make(map[string]*Session),
		Logger:    logger,
		StartTime: time.Now(),
	}
}
