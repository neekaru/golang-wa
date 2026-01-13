package client

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
)

// Client represents a WhatsApp client
type Client struct {
	ID              string
	WhatsmeowClient *whatsmeow.Client
	Container       *sqlstore.Container
	Status          ClientStatus
	manager         *ClientManager

	reconnectAttempts int
	lastReconnectTime time.Time
	lastActivityTime  time.Time

	// Mutex for protecting client state
	mu sync.Mutex
}

// Connect connects the client to WhatsApp
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update last activity time
	c.lastActivityTime = time.Now()

	if c.WhatsmeowClient.IsConnected() {
		return nil
	}

	c.Status = StatusConnecting
	c.manager.DispatchEvent(NewStatusEvent(c.ID, c.Status))

	err := c.WhatsmeowClient.Connect()
	if err != nil {
		c.Status = StatusError
		c.manager.DispatchEvent(NewStatusEvent(c.ID, c.Status))
		c.manager.logger.Printf("Error connecting client %s: %v", c.ID, err)
		return err
	}

	c.Status = StatusConnected
	c.manager.DispatchEvent(NewStatusEvent(c.ID, c.Status))
	c.reconnectAttempts = 0

	return nil
}

// Disconnect disconnects the client from WhatsApp
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update last activity time
	c.lastActivityTime = time.Now()

	if !c.WhatsmeowClient.IsConnected() {
		return
	}

	c.WhatsmeowClient.Disconnect()
	c.Status = StatusDisconnected
	c.manager.DispatchEvent(NewStatusEvent(c.ID, c.Status))
}

// Reconnect attempts to reconnect the client with exponential backoff
func (c *Client) Reconnect() {
	c.mu.Lock()

	// Update last activity time
	c.lastActivityTime = time.Now()

	// Calculate backoff time
	backoffSeconds := math.Min(30, math.Pow(2, float64(c.reconnectAttempts)))
	backoffDuration := time.Duration(backoffSeconds) * time.Second

	// Check if we need to wait before reconnecting
	timeSinceLastReconnect := time.Since(c.lastReconnectTime)
	if timeSinceLastReconnect < backoffDuration {
		waitTime := backoffDuration - timeSinceLastReconnect
		c.mu.Unlock()

		c.manager.logger.Printf("Waiting %v before reconnecting client %s (attempt %d)",
			waitTime, c.ID, c.reconnectAttempts+1)
		time.Sleep(waitTime)
		c.mu.Lock()
	}

	c.lastReconnectTime = time.Now()
	c.reconnectAttempts++
	c.mu.Unlock()

	c.manager.logger.Printf("Attempting to reconnect client %s (attempt %d)",
		c.ID, c.reconnectAttempts)

	err := c.Connect()
	if err != nil {
		c.manager.logger.Printf("Reconnection attempt %d for client %s failed: %v",
			c.reconnectAttempts, c.ID, err)
	} else {
		c.manager.logger.Printf("Successfully reconnected client %s after %d attempts",
			c.ID, c.reconnectAttempts)
	}
}

// IsLoggedIn returns whether the client is logged in
func (c *Client) IsLoggedIn() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Status == StatusLoggedIn
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.WhatsmeowClient.IsConnected()
}

// NeedsQR returns whether the client needs a QR code for login
func (c *Client) NeedsQR() bool {
	return c.WhatsmeowClient.Store.ID == nil
}

// handleWhatsmeowEvent handles events from the whatsmeow client
func (c *Client) handleWhatsmeowEvent(evt interface{}) {
	c.mu.Lock()
	// Update last activity time
	c.lastActivityTime = time.Now()
	c.mu.Unlock()

	switch e := evt.(type) {
	case *events.Connected:
		c.mu.Lock()
		// When connected, we're also logged in
		c.Status = StatusLoggedIn
		c.mu.Unlock()
		c.manager.DispatchEvent(NewStatusEvent(c.ID, c.Status))
		c.manager.logger.Printf("Client %s connected and logged in", c.ID)

	case *events.LoggedOut:
		// 'e' already has type *events.LoggedOut inside this case of the type switch.
		lo := e
		if lo.OnConnect {
			c.manager.logger.Printf("Client %s logged out on connect; reason=%s", c.ID, lo.Reason.String())
		} else {
			c.manager.logger.Printf("Client %s logged out (stream error). Reason not provided in stream:error", c.ID)
		}

		c.mu.Lock()
		c.Status = StatusLoggedOut
		c.mu.Unlock()
		c.manager.DispatchEvent(NewStatusEvent(c.ID, c.Status))

	case *events.Disconnected:
		c.mu.Lock()
		wasConnected := c.Status == StatusConnected || c.Status == StatusLoggedIn
		c.Status = StatusDisconnected
		c.mu.Unlock()

		c.manager.DispatchEvent(NewStatusEvent(c.ID, c.Status))
		c.manager.logger.Printf("Client %s disconnected", c.ID)

		// Only attempt to reconnect if we were previously connected
		if wasConnected {
			go c.Reconnect()
		}

	case *events.StreamError:
		c.manager.logger.Printf("Client %s stream error: %v", c.ID, e)
		c.manager.DispatchEvent(NewErrorEvent(c.ID, fmt.Sprintf("Stream error: %v", e)))

	case *events.QR:
		c.manager.logger.Printf("Client %s received QR code", c.ID)
		c.manager.DispatchEvent(NewQREvent(c.ID, e))
	}

	// Dispatch the raw event as well
	c.manager.DispatchEvent(NewRawEvent(c.ID, evt))
}
