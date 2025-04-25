package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
)

type Session struct {
	Client       *whatsmeow.Client
	Container    *sqlstore.Container
	User         string
	Phone        string
	IsLoggedIn   bool
	LatestQRCode string       // Store the latest QR code
	QRLock       sync.RWMutex // Lock to protect access to LatestQRCode
}

var (
	sessions     = make(map[string]*Session)
	sessionsLock = sync.RWMutex{}
	startTime    = time.Now() // Track startup time for health checks
	appLogger    *log.Logger  // Application logger
)

// setupLogging configures the application logging
func setupLogging() (*log.Logger, error) {
	// Ensure logs directory exists
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %v", err)
	}

	// Create log file with timestamp in filename
	logFilePath := filepath.Join(logDir, fmt.Sprintf("whatsapp-api-%s.log", time.Now().Format("2006-01-02")))
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Create multi-writer to log to both file and console
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)
	logger.Printf("Logging initialized to %s", logFilePath)

	// Print log location for easier access
	fmt.Printf("Logs are being written to: %s\n", logFilePath)

	return logger, nil
}

func restoreSession(user string) (*Session, error) {
	dbPath := "data/" + user + ".db"

	// Create a logger specifically for this database connection
	dbLogger := waLog.Stdout("Database-"+user, "INFO", true)
	if appLogger != nil {
		appLogger.Printf("Creating/restoring session for user: %s at %s", user, dbPath)
	}

	container, err := sqlstore.New("sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLogger)
	if err != nil {
		if appLogger != nil {
			appLogger.Printf("Database error for user %s: %v", user, err)
		}
		return nil, fmt.Errorf("database error: %v", err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		if appLogger != nil {
			appLogger.Printf("Device error for user %s: %v", user, err)
		}
		return nil, fmt.Errorf("device error: %v", err)
	}

	store.SetOSInfo("Linux", store.GetWAVersion())
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()

	// Configure client with proper logging
	clientLogger := waLog.Stdout("WhatsApp-"+user, "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLogger)

	session := &Session{
		Client:     client,
		Container:  container,
		User:       user,
		IsLoggedIn: false,
	}

	// Add connection event handler that properly updates the session state
	client.AddEventHandler(func(evt interface{}) {
		switch e := evt.(type) {
		case *events.Connected:
			if appLogger != nil {
				appLogger.Printf("User %s connected to WhatsApp", user)
			}
			session.IsLoggedIn = true
		case *events.LoggedOut:
			if appLogger != nil {
				appLogger.Printf("User %s logged out from WhatsApp", user)
			}
			session.IsLoggedIn = false
		case *events.PushName:
			if appLogger != nil {
				appLogger.Printf("User %s push name updated: %s", user, e)
			}
		case *events.StreamError:
			if appLogger != nil {
				appLogger.Printf("User %s stream error: %v", user, e)
			}
		case *events.QR:
			if appLogger != nil {
				appLogger.Printf("User %s received new QR code", user)
			}
		}
	})

	// Attempt to restore connection if device is already registered
	if client.Store.ID != nil {
		if appLogger != nil {
			appLogger.Printf("Device is registered for user %s, attempting to connect", user)
		}

		// Try to connect with retry logic for transient errors
		err = connectWithRetry(client, user)
		if err == nil {
			session.IsLoggedIn = true
			if appLogger != nil {
				appLogger.Printf("Successfully connected existing session for user: %s", user)
			}
		} else if appLogger != nil {
			appLogger.Printf("Failed to connect existing session for user %s: %v", user, err)
			// Don't return error here, as we want to return the session anyway
			// The client can try to reconnect later
		}
	} else if appLogger != nil {
		appLogger.Printf("Device not yet registered for user %s, QR code needed", user)
	}

	return session, nil
}

// Helper function to connect with retry logic
func connectWithRetry(client *whatsmeow.Client, user string) error {
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		// If client is already connected, disconnect first to avoid "already connected" errors
		if client.IsConnected() {
			if appLogger != nil {
				appLogger.Printf("Client for user %s is already connected, disconnecting first", user)
			}
			client.Disconnect()
			time.Sleep(500 * time.Millisecond)
		}

		err = client.Connect()
		if err == nil {
			return nil // Successfully connected
		}

		if strings.Contains(err.Error(), "websocket is already connected") {
			// Special handling for this common error
			if appLogger != nil {
				appLogger.Printf("Got 'already connected' error for user %s, trying again after disconnect (attempt %d/%d)",
					user, i+1, maxRetries)
			}
			client.Disconnect()
			time.Sleep(1 * time.Second) // Longer wait after this specific error
		} else {
			// For other errors, try again with shorter wait
			if appLogger != nil {
				appLogger.Printf("Connection error for user %s: %v (attempt %d/%d)",
					user, err, i+1, maxRetries)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	return err // Return the last error
}

func findSessionByUser(user string) (*Session, bool) {
	sessionsLock.RLock()
	defer sessionsLock.RUnlock()

	// First check in-memory sessions
	for _, sess := range sessions {
		if sess.User == user {
			return sess, true
		}
	}

	// If not found in memory, try to restore from database
	sess, err := restoreSession(user)
	if err != nil {
		if appLogger != nil {
			appLogger.Printf("Failed to restore session for user %s: %v", user, err)
		}
		return nil, false
	}

	// Add restored session to in-memory map
	sessionsLock.Lock()
	sessions[user] = sess
	sessionsLock.Unlock()

	return sess, true
}

func main() {
	var err error

	// Set up logging
	appLogger, err = setupLogging()
	if err != nil {
		fmt.Printf("Failed to set up logging: %v\n", err)
		// Continue with console logging only
		appLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	appLogger.Println("Starting WhatsApp API service")

	// Set Gin to release mode in production
	// gin.SetMode(gin.ReleaseMode)

	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		appLogger.Fatalf("Failed to create data directory: %v", err)
	}
	appLogger.Println("Ensured data directory exists")

	// Database directory check
	files, err := os.ReadDir("data")
	if err == nil {
		// Restore all existing sessions
		appLogger.Println("Checking for existing sessions in data directory")
		sessionCount := 0
		for _, file := range files {
			if !file.IsDir() && len(file.Name()) > 3 && file.Name()[len(file.Name())-3:] == ".db" {
				user := file.Name()[:len(file.Name())-3] // Remove .db extension
				appLogger.Printf("Found database for user: %s", user)

				if sess, err := restoreSession(user); err == nil {
					sessionsLock.Lock()
					sessions[user] = sess
					sessionsLock.Unlock()
					sessionCount++
					appLogger.Printf("Restored session for user: %s", user)
				} else {
					appLogger.Printf("Failed to restore session for user %s: %v", user, err)
				}
			}
		}
		appLogger.Printf("Restored %d sessions", sessionCount)
	} else {
		appLogger.Printf("Failed to read data directory: %v", err)
	}

	r := gin.Default()

	// Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	corsConfig.ExposeHeaders = []string{"Content-Length", "Content-Type"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 12 * time.Hour
	r.Use(cors.New(corsConfig))

	// Set up gin to log to the same log file
	gin.DefaultWriter = io.MultiWriter(os.Stdout, appLogger.Writer())
	gin.DefaultErrorWriter = io.MultiWriter(os.Stderr, appLogger.Writer())

	// Enhanced health check endpoint for Docker
	r.GET("/", func(c *gin.Context) {
		uptime := time.Since(startTime).String()
		sessionCount := len(sessions)
		c.JSON(http.StatusOK, gin.H{
			"status":        "ok",
			"uptime":        uptime,
			"session_count": sessionCount,
			"version":       "1.0.0",
		})
	})

	// Health check endpoint with detailed status
	r.GET("/health", func(c *gin.Context) {
		uptime := time.Since(startTime).String()

		// Use a try-lock approach to avoid deadlock during session initialization
		// If we can't acquire the lock within a short time, proceed with partial data
		var sessionCount, activeCount int
		lockChan := make(chan struct{})

		go func() {
			sessionsLock.RLock()
			defer sessionsLock.RUnlock()

			sessionCount = len(sessions)
			for _, sess := range sessions {
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
			appLogger.Printf("Health check timed out waiting for sessions lock")
		}

		// Log health check access for debugging
		appLogger.Printf("Health check requested from %s", c.ClientIP())

		// Always return 200 OK status
		c.JSON(http.StatusOK, gin.H{
			"status":          "ok",
			"uptime":          uptime,
			"total_sessions":  sessionCount,
			"active_sessions": activeCount,
			"timestamp":       time.Now().Format(time.RFC3339),
		})
	})

	// Also add a /health/ endpoint with trailing slash to handle both versions
	r.GET("/health/", func(c *gin.Context) {
		uptime := time.Since(startTime).String()

		// Use a try-lock approach to avoid deadlock during session initialization
		// If we can't acquire the lock within a short time, proceed with partial data
		var sessionCount, activeCount int
		lockChan := make(chan struct{})

		go func() {
			sessionsLock.RLock()
			defer sessionsLock.RUnlock()

			sessionCount = len(sessions)
			for _, sess := range sessions {
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
			appLogger.Printf("Health check (trailing slash) timed out waiting for sessions lock")
		}

		// Log health check access for debugging
		appLogger.Printf("Health check (with trailing slash) requested from %s", c.ClientIP())

		c.JSON(http.StatusOK, gin.H{
			"status":          "ok",
			"uptime":          uptime,
			"total_sessions":  sessionCount,
			"active_sessions": activeCount,
			"timestamp":       time.Now().Format(time.RFC3339),
		})
	})

	r.POST("/wa/add", handleAddSession)
	r.GET("/wa/qr-image", handleQRImage)
	r.POST("/wa/status", handleStatus)
	r.GET("/wa/status", handleStatus) // Add GET method for status
	r.POST("/wa/restart", handleRestart)
	r.POST("/wa/logout", handleLogout)

	r.POST("/send", handleSendMessage)
	r.POST("/send/file", func(c *gin.Context) { sendMediaHandler(c, "file") })
	r.POST("/send/image", func(c *gin.Context) { sendMediaHandler(c, "image") })
	r.POST("/send/video", func(c *gin.Context) { sendMediaHandler(c, "video") })

	r.POST("/msg/read", handleMarkRead)

	// Only use port 8080, Caddy will handle port 80
	srv := &http.Server{Addr: ":8080", Handler: r}
	go func() {
		appLogger.Println("🚀 WhatsApp bot running on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Printf("Server error: %v\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Println("🚫 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Printf("Server forced to shutdown: %v\n", err)
	}

	appLogger.Println("Server exited")
}

func handleSendMessage(c *gin.Context) {
	var req struct {
		User        string `json:"user"`
		PhoneNumber string `json:"phone_number"`
		Message     string `json:"message"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	sess, exists := findSessionByUser(req.User)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session not found"})
		return
	}

	// Ensure client is connected before sending
	if !sess.Client.IsConnected() {
		err := sess.Client.Connect()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect: " + err.Error()})
			return
		}
	}

	// Create recipient JID
	recipient := types.JID{
		User:   req.PhoneNumber,
		Server: "s.whatsapp.net",
	}

	// Create message and send
	msg := &waProto.Message{
		Conversation: proto.String(req.Message),
	}

	opts := whatsmeow.SendRequestExtra{
		ID: whatsmeow.GenerateMessageID(),
	}

	_, err := sess.Client.SendMessage(context.Background(), recipient, msg, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message: " + err.Error()})
		return
	}

	// Log successful message send
	appLogger.Printf("Message sent successfully to %s from user %s", recipient.String(), req.User)

	c.JSON(http.StatusOK, gin.H{"msg": "Message sent successfully"})
}

func handleMarkRead(c *gin.Context) {
	var req struct {
		User      string   `json:"user"`
		MessageID []string `json:"message_ids"`
		FromJID   string   `json:"from_jid"`
		ToJID     string   `json:"to_jid"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	sess, exists := findSessionByUser(req.User)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session not found"})
		return
	}

	// Convert string message IDs to types.MessageID
	messageIDs := make([]types.MessageID, len(req.MessageID))
	for i, id := range req.MessageID {
		messageIDs[i] = types.MessageID(id)
	}

	fromJID := types.JID{User: req.FromJID, Server: "s.whatsapp.net"}
	toJID := types.JID{User: req.ToJID, Server: "s.whatsapp.net"}

	err := sess.Client.MarkRead(messageIDs, time.Now(), fromJID, toJID, types.ReceiptTypeRead)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "Messages marked as read"})
}

func sendMediaHandler(c *gin.Context, mediaType string) {
	var req struct {
		User        string `json:"user"`
		PhoneNumber string `json:"phone_number"`
		Media       string `json:"media"`
		URL         string `json:"url"`
		Caption     string `json:"caption"`
		FileName    string `json:"file_name"` // Optional filename parameter
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	sess, exists := findSessionByUser(req.User)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session not found"})
		return
	}

	// Ensure client is connected before sending
	if !sess.Client.IsConnected() {
		err := sess.Client.Connect()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect: " + err.Error()})
			return
		}
	}

	recipient := types.JID{
		User:   req.PhoneNumber,
		Server: "s.whatsapp.net",
	}

	var media []byte
	var mimeType string
	var fileName string

	// Set filename if provided
	if req.FileName != "" {
		fileName = req.FileName
	}

	// Check if URL or base64 media is provided
	if req.URL != "" {
		// Download media from URL
		httpResp, err := http.Get(req.URL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to download media from URL: " + err.Error()})
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to download media, status: " + httpResp.Status})
			return
		}

		media, err = io.ReadAll(httpResp.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read media from URL: " + err.Error()})
			return
		}

		mimeType = httpResp.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = http.DetectContentType(media)
		}

		// Extract filename from URL if not provided
		if fileName == "" {
			// Parse URL to extract filename
			parsedURL, err := url.Parse(req.URL)
			if err == nil {
				// Get the last part of the path
				parts := strings.Split(parsedURL.Path, "/")
				if len(parts) > 0 {
					urlFileName := parts[len(parts)-1]
					// Remove query parameters if present
					urlFileName = strings.Split(urlFileName, "?")[0]
					// Use it if it looks like a valid filename
					if urlFileName != "" && !strings.HasSuffix(urlFileName, "/") {
						fileName = urlFileName
						appLogger.Printf("Extracted filename from URL: %s", fileName)
					}
				}
			}
		}

		// Try to get filename from Content-Disposition header if still not found
		if fileName == "" {
			contentDisposition := httpResp.Header.Get("Content-Disposition")
			if contentDisposition != "" {
				if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
					if fn, ok := params["filename"]; ok && fn != "" {
						fileName = fn
						appLogger.Printf("Extracted filename from Content-Disposition: %s", fileName)
					}
				}
			}
		}
	} else if req.Media != "" {
		// Decode base64 media
		var err error
		media, err = base64.StdEncoding.DecodeString(req.Media)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media format"})
			return
		}
		mimeType = http.DetectContentType(media)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either media or URL must be provided"})
		return
	}

	var waMediaType whatsmeow.MediaType
	switch mediaType {
	case "image":
		waMediaType = whatsmeow.MediaImage
	case "video":
		waMediaType = whatsmeow.MediaVideo
	case "file":
		waMediaType = whatsmeow.MediaDocument
	}

	uploaded, err := sess.Client.Upload(context.Background(), media, waMediaType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload media"})
		return
	}

	var msg waE2E.Message
	switch mediaType {
	case "image":
		msg = waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String(req.Caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(media))),
			},
		}
	case "video":
		msg = waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:       proto.String(req.Caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(media))),
			},
		}
	case "file":
		msg = waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				Caption:       proto.String(req.Caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(media))),
				FileName:      proto.String(fileName),
			},
		}
	}

	opts := whatsmeow.SendRequestExtra{
		ID: types.MessageID(fmt.Sprintf("%d", time.Now().UnixNano())),
	}

	_, err = sess.Client.SendMessage(context.Background(), recipient, &msg, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send media message: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":       mediaType + " sent successfully",
		"file_name": fileName,
	})
}

func handleRestart(c *gin.Context) {
	user := c.Query("user")
	if user == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user"})
		return
	}

	appLogger.Printf("Restarting session for user: %s", user)

	// First disconnect existing session if it exists
	if oldSess, exists := findSessionByUser(user); exists {
		appLogger.Printf("Disconnecting existing session for user: %s", user)
		// Safe disconnect with retry
		if oldSess.Client.IsConnected() {
			oldSess.Client.Disconnect()
			// Give it a moment to properly disconnect
			time.Sleep(500 * time.Millisecond)
		}

		// Remove from memory to force database restoration
		sessionsLock.Lock()
		delete(sessions, user)
		sessionsLock.Unlock()
	}

	// Attempt to restore session from database
	sess, err := restoreSession(user)
	if err != nil {
		appLogger.Printf("Failed to restore session for user %s: %v", user, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore session: " + err.Error()})
		return
	}

	appLogger.Printf("Session restored from database for user: %s", user)

	// Add restored session to memory
	sessionsLock.Lock()
	sessions[user] = sess
	sessionsLock.Unlock()

	// Connect the restored session with retry logic
	err = sess.Client.Connect()
	if err != nil {
		// Try to handle specific error types
		if strings.Contains(err.Error(), "websocket is already connected") {
			appLogger.Printf("Got 'already connected' error for %s, trying to disconnect and reconnect", user)
			// Force disconnect and try again after a delay
			sess.Client.Disconnect()
			time.Sleep(1 * time.Second)
			err = sess.Client.Connect()
			if err != nil {
				appLogger.Printf("Failed to connect after retry for user %s: %v", user, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to connect after retry: " + err.Error(),
					"status": map[string]interface{}{
						"logged_in": sess.IsLoggedIn,
						"connected": sess.Client.IsConnected(),
						"user":      user,
					},
				})
				return
			}
		} else {
			appLogger.Printf("Failed to connect for user %s: %v", user, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to connect restored session: " + err.Error(),
				"status": map[string]interface{}{
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

	appLogger.Printf("Session successfully reconnected for user: %s (logged_in=%v, connected=%v)",
		user, isLoggedIn, isConnected)

	c.JSON(http.StatusOK, gin.H{
		"msg": "Session restored and connected successfully",
		"status": map[string]interface{}{
			"logged_in": isLoggedIn,
			"connected": isConnected,
			"user":      user,
			"needs_qr":  !isLoggedIn || !isConnected,
		},
	})
}

func handleStatus(c *gin.Context) {
	user := c.Query("user")
	if user == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user"})
		return
	}

	sess, exists := findSessionByUser(user)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session not found"})
		return
	}

	// Get connection details
	isLoggedIn := sess.IsLoggedIn
	isConnected := sess.Client.IsConnected()

	// Log the status check
	appLogger.Printf("Status check for user %s: logged_in=%v, connected=%v",
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

func handleAddSession(c *gin.Context) {
	var req struct {
		User string `json:"user"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Ensure data directory exists
	os.MkdirAll("data", 0755)

	dbPath := "data/" + req.User + ".db"

	// Initialize the database connection
	dbLog := waLog.Stdout("Database", "INFO", true)
	container, err := sqlstore.New("sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error: " + err.Error()})
		return
	}

	// Get the device store from the database
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Device error: " + err.Error()})
		return
	}

	// Create the client, but don't connect yet
	store.SetOSInfo("Linux", store.GetWAVersion())
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	// Create a new session and store it in the sessions map
	session := &Session{
		Client:     client,
		Container:  container,
		User:       req.User,
		IsLoggedIn: false,
	}

	// Add the session to the sessions map
	sessionsLock.Lock()
	sessions[req.User] = session
	sessionsLock.Unlock()

	// Inform the client that the session was created successfully, but QR generation is pending
	c.JSON(http.StatusOK, gin.H{"msg": "Session created. Please request QR code using /wa/qr-image"})
}

func handleQRImage(c *gin.Context) {
	user := c.Query("user")
	if user == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user"})
		return
	}

	sess, exists := findSessionByUser(user)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session not found"})
		return
	}

	// Check both logged_in and connection status
	if sess.IsLoggedIn && sess.Client.IsConnected() {
		appLogger.Printf("User %s is already logged in and connected, no QR code needed", user)
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

	// Always disconnect first to avoid "websocket is already connected" error
	// This is safe to call even if not connected
	if sess.Client.IsConnected() {
		appLogger.Printf("Disconnecting existing connection for user %s before generating QR", user)
		sess.Client.Disconnect()
		// Small delay to ensure disconnection is complete
		time.Sleep(500 * time.Millisecond)
	}

	// Reset login state to be safe
	sess.IsLoggedIn = false

	// Set up a channel to receive the QR code
	qrCodeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start the client connection and QR code generation in a goroutine
	go func() {
		client := sess.Client

		// Set up event handlers before connecting
		qrChan, _ := client.GetQRChannel(context.Background())

		// Connect the client with error handling
		err := client.Connect()
		if err != nil {
			// Try to handle specific error types
			if strings.Contains(err.Error(), "websocket is already connected") {
				appLogger.Printf("Got 'already connected' error for %s, trying to disconnect and reconnect", user)
				// Force disconnect and try again after a delay
				client.Disconnect()
				time.Sleep(1 * time.Second)
				err = client.Connect()
				if err != nil {
					errorChan <- fmt.Errorf("failed to connect client after retry: %v", err)
					return
				}
			} else {
				errorChan <- fmt.Errorf("failed to connect client: %v", err)
				return
			}
		}

		// Add connection event handler
		client.AddEventHandler(func(evt interface{}) {
			switch evt.(type) {
			case *events.Connected:
				sess.IsLoggedIn = true
				appLogger.Printf("User %s connection state changed to: connected", user)
			case *events.LoggedOut:
				sess.IsLoggedIn = false
				appLogger.Printf("User %s connection state changed to: logged out", user)
			}
		})

		// Wait for QR code
		if qrChan != nil {
			select {
			case evt := <-qrChan:
				if evt.Code != "" {
					sess.QRLock.Lock()
					sess.LatestQRCode = evt.Code
					sess.QRLock.Unlock()

					appLogger.Printf("Generated QR code for user %s", user)

					// Generate QR code image
					qr, err := qrcode.New(evt.Code, qrcode.Medium)
					if err != nil {
						errorChan <- fmt.Errorf("failed to generate QR code: %v", err)
						return
					}

					// Convert QR code to PNG bytes
					png, err := qr.PNG(256)
					if err != nil {
						errorChan <- fmt.Errorf("failed to generate PNG: %v", err)
						return
					}

					// Convert to base64
					qrBase64 := base64.StdEncoding.EncodeToString(png)
					qrCodeChan <- qrBase64
				} else {
					errorChan <- fmt.Errorf("received empty QR code")
				}
			case <-time.After(30 * time.Second):
				errorChan <- fmt.Errorf("timed out waiting for QR code generation")
			}
		} else {
			errorChan <- fmt.Errorf("failed to create QR channel")
		}
	}()

	// Wait for either the QR code or an error
	select {
	case qrCode := <-qrCodeChan:
		c.JSON(http.StatusOK, gin.H{"qrcode": "data:image/png;base64," + qrCode})
	case err := <-errorChan:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	case <-time.After(60 * time.Second):
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "QR code not available after waiting for 60 seconds"})
	}
}

// Add more handlers for logout, send message, etc.

func handleLogout(c *gin.Context) {
	var req struct {
		User string `json:"user"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "Logout process started"})

	go func() {
		sessionsLock.Lock()
		defer sessionsLock.Unlock()

		var sessionKey string
		var sess *Session

		for key, s := range sessions {
			if s.User == req.User {
				sessionKey = key
				sess = s
				break
			}
		}

		if sess == nil {
			fmt.Printf("Logout: No session found for user %s\n", req.User)
			return
		}

		// Step 1: Logout and disconnect client safely
		if sess.Client != nil {
			if err := sess.Client.Logout(); err != nil {
				fmt.Printf("Error during logout for %s: %v\n", req.User, err)
			} else {
				fmt.Printf("Successfully logged out %s\n", req.User)
			}
			sess.Client.Disconnect()
		}

		// Step 2: Close database connection
		if sess.Container != nil {
			sess.Container.Close()
		}

		// Step 3: Delete database file
		dbFile := "data/" + req.User + ".db"
		if err := os.Remove(dbFile); err != nil {
			fmt.Printf("Error deleting database file for %s: %v\n", req.User, err)
		} else {
			fmt.Printf("Successfully deleted database file for %s\n", req.User)
		}

		// Step 4: Remove from sessions map
		delete(sessions, sessionKey)
	}()
}
