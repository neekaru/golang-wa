package session

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/neekaru/whatsappgo-bot/internal/app"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// KeyedMutex is a string-keyed mutex for locking operations on specific keys
type KeyedMutex struct {
	mutexes sync.Map
}

// Lock locks the mutex for the given key
func (m *KeyedMutex) Lock(key string) {
	value, _ := m.mutexes.LoadOrStore(key, &sync.Mutex{})
	mtx := value.(*sync.Mutex)
	mtx.Lock()
}

// Unlock unlocks the mutex for the given key
func (m *KeyedMutex) Unlock(key string) {
	value, ok := m.mutexes.Load(key)
	if !ok {
		panic(fmt.Sprintf("unlock of unlocked mutex for key %s", key))
	}
	mtx := value.(*sync.Mutex)
	mtx.Unlock()
}

// Global mutex for session restoration to prevent concurrent restoration of the same session
var sessionRestorationMutex = &KeyedMutex{}

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
	// Check if the client already exists in the ClientManager
	clientManager := s.app.GetClientManager()
	if clientManager.ClientExists(user) {
		// Client already exists in the ClientManager, create a legacy Session for it
		whatsappClient, _ := clientManager.GetClient(user)
		session := &app.Session{
			Client:     whatsappClient.WhatsmeowClient,
			Container:  whatsappClient.Container,
			User:       user,
			IsLoggedIn: whatsappClient.IsLoggedIn(),
		}
		return session, nil
	}

	// Client doesn't exist, restore it from the database
	dbPath := "data/" + user + ".db"

	// Create a context with timeout to prevent indefinite blocking
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a logger specifically for this database connection
	dbLogger := waLog.Stdout("Database-"+user, "INFO", true)
	s.app.Logger.Printf("Creating/restoring session for user: %s at %s", user, dbPath)

	// Use a channel to handle the database operation with timeout
	type dbResult struct {
		container *sqlstore.Container
		err       error
	}

	resultChan := make(chan dbResult, 1)

	go func() {
		// Try to open the database with compatibility for old format
		container, err := sqlstore.New("sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLogger)
		resultChan <- dbResult{container, err}
	}()

	// Wait for either the result or timeout
	var container *sqlstore.Container
	var err error

	select {
	case result := <-resultChan:
		container = result.container
		err = result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("database operation timed out: %v", ctx.Err())
	}

	if err != nil {
		s.app.Logger.Printf("Database error for user %s: %v", user, err)
		return nil, fmt.Errorf("database error: %v", err)
	}

	// Get device with timeout
	deviceChan := make(chan struct {
		device *store.Device
		err    error
	}, 1)

	go func() {
		deviceStore, err := container.GetFirstDevice()
		deviceChan <- struct {
			device *store.Device
			err    error
		}{deviceStore, err}
	}()

	var deviceStore *store.Device

	select {
	case result := <-deviceChan:
		deviceStore = result.device
		err = result.err
	case <-ctx.Done():
		// Close the container before returning
		if container != nil {
			container.Close()
		}
		return nil, fmt.Errorf("device operation timed out: %v", ctx.Err())
	}

	if err != nil {
		s.app.Logger.Printf("Device error for user %s: %v", user, err)
		// Close the container before returning
		if container != nil {
			container.Close()
		}
		return nil, fmt.Errorf("device error: %v", err)
	}

	store.SetOSInfo("Linux", store.GetWAVersion())
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()

	// Configure client with proper logging
	clientLogger := waLog.Stdout("WhatsApp-"+user, "INFO", true)
	whatsmeowClient := whatsmeow.NewClient(deviceStore, clientLogger)

	// Add the client to the ClientManager
	_, err = clientManager.AddClient(user, container, whatsmeowClient)
	if err != nil {
		s.app.Logger.Printf("Error adding client to ClientManager: %v", err)
		// Close the container before returning
		if container != nil {
			container.Close()
		}
		return nil, fmt.Errorf("error adding client to ClientManager: %v", err)
	}

	// Create a legacy Session for backward compatibility
	session := &app.Session{
		Client:     whatsmeowClient,
		Container:  container,
		User:       user,
		IsLoggedIn: false,
	}

	// Attempt to connect if device is already registered
	if whatsmeowClient.Store.ID != nil {
		s.app.Logger.Printf("Device is registered for user %s, attempting to connect", user)

		// Get the client from the ClientManager
		whatsappClient, exists := clientManager.GetClient(user)
		if !exists {
			s.app.Logger.Printf("Client not found in ClientManager for user %s", user)
		} else {
			// Use the ClientManager's client to connect
			err = whatsappClient.Connect()
			if err == nil {
				session.IsLoggedIn = true
				s.app.Logger.Printf("Successfully connected existing session for user: %s", user)
			} else {
				s.app.Logger.Printf("Failed to connect existing session for user %s: %v", user, err)
				// Don't return error here, as we want to return the session anyway
			}
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
	// Use a mutex to prevent concurrent restoration of the same session
	sessionRestorationMutex.Lock(user)
	defer sessionRestorationMutex.Unlock(user)

	// Check if the client exists in the ClientManager
	clientManager := s.app.GetClientManager()
	if clientManager.ClientExists(user) {
		// Client exists in the ClientManager, create a legacy Session for it
		whatsappClient, _ := clientManager.GetClient(user)
		session := &app.Session{
			Client:     whatsappClient.WhatsmeowClient,
			Container:  whatsappClient.Container,
			User:       user,
			IsLoggedIn: whatsappClient.IsLoggedIn(),
		}
		return session, true
	}

	// First check in-memory sessions with read lock only
	s.app.SessionsLock.RLock()
	for _, sess := range s.app.Sessions {
		if sess.User == user {
			// Found in memory, return immediately
			s.app.SessionsLock.RUnlock()
			return sess, true
		}
	}
	// Release read lock before database operations
	s.app.SessionsLock.RUnlock()

	// Check one more time with a write lock to ensure atomicity
	s.app.SessionsLock.Lock()
	if existingSession, exists := s.app.Sessions[user]; exists {
		s.app.SessionsLock.Unlock()
		return existingSession, true
	}
	s.app.SessionsLock.Unlock()

	// If not found in memory, try to restore from database
	s.app.Logger.Printf("Session not found in memory, restoring from database for user: %s", user)
	sess, err := s.RestoreSession(user)
	if err != nil {
		s.app.Logger.Printf("Failed to restore session for user %s: %v", user, err)
		return nil, false
	}

	// Add restored session to in-memory map with a separate write lock
	s.app.SessionsLock.Lock()
	// Final check if another thread somehow added the session while we were restoring
	if existingSession, exists := s.app.Sessions[user]; exists {
		s.app.SessionsLock.Unlock()
		// Close the resources we just created since we won't be using them
		if sess.Container != nil {
			sess.Container.Close()
		}
		if sess.Client != nil && sess.Client.IsConnected() {
			sess.Client.Disconnect()
		}
		return existingSession, true
	}
	// Add our newly restored session
	s.app.Sessions[user] = sess
	s.app.SessionsLock.Unlock()

	return sess, true
}

// AddSession creates a new WhatsApp session
func (s *Service) AddSession(user string) (*app.Session, error) {
	// Check if the client already exists in the ClientManager
	clientManager := s.app.GetClientManager()
	if clientManager.ClientExists(user) {
		// Client already exists, create a legacy Session for it
		whatsappClient, _ := clientManager.GetClient(user)
		session := &app.Session{
			Client:     whatsappClient.WhatsmeowClient,
			Container:  whatsappClient.Container,
			User:       user,
			IsLoggedIn: whatsappClient.IsLoggedIn(),
		}
		return session, nil
	}

	dbPath := "data/" + user + ".db"

	// Initialize the database connection
	dbLog := waLog.Stdout("Database", "INFO", true)
	container, err := sqlstore.New("sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("db error: %v", err)
	}

	// Get the device store from the database
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("device error: %v", err)
	}

	// Create the client, but don't connect yet
	store.SetOSInfo("Linux", store.GetWAVersion())
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	whatsmeowClient := whatsmeow.NewClient(deviceStore, waLog.Noop)

	// Add the client to the ClientManager
	_, err = clientManager.AddClient(user, container, whatsmeowClient)
	if err != nil {
		s.app.Logger.Printf("Error adding client to ClientManager: %v", err)
		// Close the container before returning
		if container != nil {
			container.Close()
		}
		return nil, fmt.Errorf("error adding client to ClientManager: %v", err)
	}

	// Create a legacy Session for backward compatibility
	session := &app.Session{
		Client:     whatsmeowClient,
		Container:  container,
		User:       user,
		IsLoggedIn: false,
	}

	// Add the session to the sessions map for backward compatibility
	s.app.SessionsLock.Lock()
	s.app.Sessions[user] = session
	s.app.SessionsLock.Unlock()

	return session, nil
}

// LogoutSession logs out a session and cleans up resources
func (s *Service) LogoutSession(user string) error {
	// Check if the client exists in the ClientManager
	clientManager := s.app.GetClientManager()
	if clientManager.ClientExists(user) {
		// Remove the client from the ClientManager
		err := clientManager.RemoveClient(user)
		if err != nil {
			s.app.Logger.Printf("Error removing client from ClientManager: %v", err)
		}
	}

	// First find the session with read lock
	s.app.SessionsLock.RLock()
	var sessionKey string
	var sess *app.Session

	for key, s := range s.app.Sessions {
		if s.User == user {
			sessionKey = key
			sess = s
			break
		}
	}
	s.app.SessionsLock.RUnlock()

	if sess == nil {
		return fmt.Errorf("no session found for user %s", user)
	}

	// Step 1: Logout and disconnect client safely
	// Note: We're not holding any locks here for these potentially slow operations
	if sess.Client != nil {
		// Only attempt to logout if the client is connected
		if sess.Client.IsConnected() {
			if err := sess.Client.Logout(); err != nil {
				s.app.Logger.Printf("Error during logout for %s: %v", user, err)
				// Continue with cleanup even if logout fails
			} else {
				s.app.Logger.Printf("Successfully logged out %s", user)
			}
			// Disconnect after logout attempt
			sess.Client.Disconnect()
		} else {
			s.app.Logger.Printf("Client for %s is not connected, skipping logout request", user)
			// No need to disconnect as it's already disconnected
		}
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
	s.app.SessionsLock.Lock()
	delete(s.app.Sessions, sessionKey)
	s.app.SessionsLock.Unlock()

	return nil
}
