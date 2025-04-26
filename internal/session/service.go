package session

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/neekaru/whatsappgo-bot/internal/app"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Service handles session-related business logic
type Service struct {
	app *app.App
}

// NewService creates a new session service
func NewService(app *app.App) *Service {
	return &Service{app: app}
}

// RestoreSession restores a session from the database
func (s *Service) RestoreSession(user string) (*app.Session, error) {
	dbPath := "data/" + user + ".db"

	// Create a logger specifically for this database connection
	dbLogger := waLog.Stdout("Database-"+user, "INFO", true)
	s.app.Logger.Printf("Creating/restoring session for user: %s at %s", user, dbPath)

	container, err := sqlstore.New("sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLogger)
	if err != nil {
		s.app.Logger.Printf("Database error for user %s: %v", user, err)
		return nil, fmt.Errorf("database error: %v", err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		s.app.Logger.Printf("Device error for user %s: %v", user, err)
		return nil, fmt.Errorf("device error: %v", err)
	}

	store.SetOSInfo("Linux", store.GetWAVersion())
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()

	// Configure client with proper logging
	clientLogger := waLog.Stdout("WhatsApp-"+user, "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLogger)

	session := &app.Session{
		Client:     client,
		Container:  container,
		User:       user,
		IsLoggedIn: false,
	}

	// Add connection event handler that properly updates the session state
	client.AddEventHandler(func(evt interface{}) {
		switch e := evt.(type) {
		case *events.Connected:
			s.app.Logger.Printf("User %s connected to WhatsApp", user)
			session.IsLoggedIn = true
		case *events.LoggedOut:
			s.app.Logger.Printf("User %s logged out from WhatsApp", user)
			session.IsLoggedIn = false
		case *events.PushName:
			s.app.Logger.Printf("User %s push name updated: %v", user, e)
		case *events.StreamError:
			s.app.Logger.Printf("User %s stream error: %v", user, e)
		case *events.QR:
			s.app.Logger.Printf("User %s received new QR code", user)
		}
	})

	// Attempt to restore connection if device is already registered
	if client.Store.ID != nil {
		s.app.Logger.Printf("Device is registered for user %s, attempting to connect", user)

		// Try to connect with retry logic for transient errors
		err = s.ConnectWithRetry(client, user)
		if err == nil {
			session.IsLoggedIn = true
			s.app.Logger.Printf("Successfully connected existing session for user: %s", user)
		} else {
			s.app.Logger.Printf("Failed to connect existing session for user %s: %v", user, err)
			// Don't return error here, as we want to return the session anyway
			// The client can try to reconnect later
		}
	} else {
		s.app.Logger.Printf("Device not yet registered for user %s, QR code needed", user)
	}

	return session, nil
}

// ConnectWithRetry attempts to connect a client with retry logic
func (s *Service) ConnectWithRetry(client *whatsmeow.Client, user string) error {
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		// If client is already connected, disconnect first to avoid "already connected" errors
		if client.IsConnected() {
			s.app.Logger.Printf("Client for user %s is already connected, disconnecting first", user)
			client.Disconnect()
			time.Sleep(500 * time.Millisecond)
		}

		err = client.Connect()
		if err == nil {
			return nil // Successfully connected
		}

		if strings.Contains(err.Error(), "websocket is already connected") {
			// Special handling for this common error
			s.app.Logger.Printf("Got 'already connected' error for user %s, trying again after disconnect (attempt %d/%d)",
				user, i+1, maxRetries)
			client.Disconnect()
			time.Sleep(1 * time.Second) // Longer wait after this specific error
		} else {
			// For other errors, try again with shorter wait
			s.app.Logger.Printf("Connection error for user %s: %v (attempt %d/%d)",
				user, err, i+1, maxRetries)
			time.Sleep(500 * time.Millisecond)
		}
	}

	return err // Return the last error
}

// FindSessionByUser finds a session by user identifier
func (s *Service) FindSessionByUser(user string) (*app.Session, bool) {
	s.app.SessionsLock.RLock()
	defer s.app.SessionsLock.RUnlock()

	// First check in-memory sessions
	for _, sess := range s.app.Sessions {
		if sess.User == user {
			return sess, true
		}
	}

	// If not found in memory, try to restore from database
	sess, err := s.RestoreSession(user)
	if err != nil {
		s.app.Logger.Printf("Failed to restore session for user %s: %v", user, err)
		return nil, false
	}

	// Add restored session to in-memory map
	s.app.SessionsLock.Lock()
	s.app.Sessions[user] = sess
	s.app.SessionsLock.Unlock()

	return sess, true
}

// AddSession creates a new WhatsApp session
func (s *Service) AddSession(user string) (*app.Session, error) {
	dbPath := "data/" + user + ".db"

	// Initialize the database connection
	dbLog := waLog.Stdout("Database", "INFO", true)
	container, err := sqlstore.New("sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("DB error: %v", err)
	}

	// Get the device store from the database
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("Device error: %v", err)
	}

	// Create the client, but don't connect yet
	store.SetOSInfo("Linux", store.GetWAVersion())
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	// Create a new session
	session := &app.Session{
		Client:     client,
		Container:  container,
		User:       user,
		IsLoggedIn: false,
	}

	// Add the session to the sessions map
	s.app.SessionsLock.Lock()
	s.app.Sessions[user] = session
	s.app.SessionsLock.Unlock()

	return session, nil
}

// LogoutSession logs out a session and cleans up resources
func (s *Service) LogoutSession(user string) error {
	s.app.SessionsLock.Lock()
	defer s.app.SessionsLock.Unlock()

	var sessionKey string
	var sess *app.Session

	for key, s := range s.app.Sessions {
		if s.User == user {
			sessionKey = key
			sess = s
			break
		}
	}

	if sess == nil {
		return fmt.Errorf("no session found for user %s", user)
	}

	// Step 1: Logout and disconnect client safely
	if sess.Client != nil {
		if err := sess.Client.Logout(); err != nil {
			s.app.Logger.Printf("Error during logout for %s: %v", user, err)
			// Continue with cleanup even if logout fails
		} else {
			s.app.Logger.Printf("Successfully logged out %s", user)
		}
		sess.Client.Disconnect()
	}

	// Step 2: Close database connection
	if sess.Container != nil {
		sess.Container.Close()
	}

	// Step 3: Delete database file
	dbFile := "data/" + user + ".db"
	if err := os.Remove(dbFile); err != nil {
		s.app.Logger.Printf("Error deleting database file for %s: %v", user, err)
		// Continue with cleanup even if file deletion fails
	} else {
		s.app.Logger.Printf("Successfully deleted database file for %s", user)
	}

	// Step 4: Remove from sessions map
	delete(s.app.Sessions, sessionKey)

	return nil
}
