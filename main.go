package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
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
)

func restoreSession(user string) (*Session, error) {
	dbPath := "data/" + user + ".db"
	dbLog := waLog.Stdout("Database", "INFO", true)
	container, err := sqlstore.New("sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("device error: %v", err)
	}

	store.SetOSInfo("Linux", store.GetWAVersion())
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	session := &Session{
		Client:     client,
		Container:  container,
		User:       user,
		IsLoggedIn: false,
	}

	// Attempt to restore connection if device is already registered
	if client.Store.ID != nil {
		err = client.Connect()
		if err == nil {
			session.IsLoggedIn = true
		}
	}

	return session, nil
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
		return nil, false
	}

	// Add restored session to in-memory map
	sessionsLock.Lock()
	sessions[user] = sess
	sessionsLock.Unlock()

	return sess, true
}

func main() {
	// Set Gin to release mode in production
	gin.SetMode(gin.ReleaseMode)

	// Ensure data directory exists
	os.MkdirAll("data", 0755)

	// Database directory check
	files, err := os.ReadDir("data")
	if err == nil {
		// Restore all existing sessions
		for _, file := range files {
			if !file.IsDir() && len(file.Name()) > 3 && file.Name()[len(file.Name())-3:] == ".db" {
				user := file.Name()[:len(file.Name())-3] // Remove .db extension
				if sess, err := restoreSession(user); err == nil {
					sessionsLock.Lock()
					sessions[user] = sess
					sessionsLock.Unlock()
					fmt.Printf("Restored session for user: %s\n", user)
				}
			}
		}
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

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
		sessionCount := len(sessions)
		
		// Check if any sessions are logged in
		activeCount := 0
		sessionsLock.RLock()
		for _, sess := range sessions {
			if sess.IsLoggedIn {
				activeCount++
			}
		}
		sessionsLock.RUnlock()
		
		c.JSON(http.StatusOK, gin.H{
			"status":           "ok",
			"uptime":           uptime,
			"total_sessions":   sessionCount,
			"active_sessions":  activeCount,
			"timestamp":        time.Now().Format(time.RFC3339),
		})
	})

	r.POST("/wa/add", handleAddSession)
	r.GET("/wa/qr-image", handleQRImage)
	r.POST("/wa/status", handleStatus)
	r.GET("/wa/status", handleStatus)  // Add GET method for status
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
		fmt.Println("🚀 WhatsApp bot running on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("🚫 Shutting down server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}
	
	fmt.Println("Server exited")
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

	// Check if number exists on WhatsApp
	resp, err := sess.Client.IsOnWhatsApp([]string{req.PhoneNumber})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check number: " + err.Error()})
		return
	}
	if len(resp) == 0 || !resp[0].IsIn {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The phone number is not registered on WhatsApp"})
		return
	}

	// Create message and send
	msg := &waProto.Message{
		Conversation: proto.String(req.Message),
	}

	opts := whatsmeow.SendRequestExtra{
		ID: types.MessageID(fmt.Sprintf("%d", time.Now().UnixNano())),
	}

	_, err = sess.Client.SendMessage(context.Background(), recipient, msg, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message: " + err.Error()})
		return
	}

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

	// Check if number exists on WhatsApp
	resp, err := sess.Client.IsOnWhatsApp([]string{req.PhoneNumber})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check number: " + err.Error()})
		return
	}
	if len(resp) == 0 || !resp[0].IsIn {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The phone number is not registered on WhatsApp"})
		return
	}

	recipient := types.JID{
		User:   req.PhoneNumber,
		Server: "s.whatsapp.net",
	}

	var media []byte
	var mimeType string

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

	var msg waProto.Message
	switch mediaType {
	case "image":
		msg = waProto.Message{
			ImageMessage: &waProto.ImageMessage{
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
		msg = waProto.Message{
			VideoMessage: &waProto.VideoMessage{
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
		msg = waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
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
	}

	opts := whatsmeow.SendRequestExtra{
		ID: types.MessageID(fmt.Sprintf("%d", time.Now().UnixNano())),
	}

	_, err = sess.Client.SendMessage(context.Background(), recipient, &msg, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send media message: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": mediaType + " sent successfully"})
}

func handleRestart(c *gin.Context) {
	user := c.Query("user")
	if user == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user"})
		return
	}

	// First disconnect existing session if it exists
	if oldSess, exists := findSessionByUser(user); exists {
		oldSess.Client.Disconnect()

		// Remove from memory to force database restoration
		sessionsLock.Lock()
		delete(sessions, user)
		sessionsLock.Unlock()
	}

	// Attempt to restore session from database
	sess, err := restoreSession(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore session: " + err.Error()})
		return
	}

	// Add restored session to memory
	sessionsLock.Lock()
	sessions[user] = sess
	sessionsLock.Unlock()

	// Connect the restored session
	err = sess.Client.Connect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect restored session: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "Session restored and connected successfully"})
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

	c.JSON(http.StatusOK, gin.H{
		"logged_in": sess.IsLoggedIn,
		"connected": sess.Client.IsConnected(),
		"user":      user,
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

	if sess.IsLoggedIn {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session is already connected"})
		return
	}

	// Set up a channel to receive the QR code
	qrCodeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start the client connection and QR code generation in a goroutine
	go func() {
		client := sess.Client

		// Set up event handlers before connecting
		qrChan, _ := client.GetQRChannel(context.Background())

		// Connect the client
		err := client.Connect()
		if err != nil {
			errorChan <- fmt.Errorf("failed to connect client: %v", err)
			return
		}

		// Add connection event handler
		client.AddEventHandler(func(evt interface{}) {
			switch evt.(type) {
			case *events.Connected:
				sess.IsLoggedIn = true
			case *events.LoggedOut:
				sess.IsLoggedIn = false
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
