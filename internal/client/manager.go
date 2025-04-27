package client

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// ClientStatus represents the current status of a client
type ClientStatus int

const (
	StatusDisconnected ClientStatus = iota
	StatusConnecting
	StatusConnected
	StatusLoggedIn
	StatusLoggedOut
	StatusError
)

// String returns a string representation of the client status
func (s ClientStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusLoggedIn:
		return "logged_in"
	case StatusLoggedOut:
		return "logged_out"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// ClientManager is a singleton that manages WhatsApp clients
type ClientManager struct {
	clients       map[string]*Client
	clientsLock   sync.RWMutex
	observers     map[string][]Observer
	observersLock sync.RWMutex
	logger        *log.Logger
	workerPool    chan func()
}

var (
	instance *ClientManager
	once     sync.Once
)

// GetInstance returns the singleton instance of ClientManager
func GetInstance() *ClientManager {
	once.Do(func() {
		instance = &ClientManager{
			clients:    make(map[string]*Client),
			observers:  make(map[string][]Observer),
			logger:     log.New(os.Stdout, "ClientManager: ", log.LstdFlags),
			workerPool: make(chan func(), 100), // Buffer size of 100 tasks
		}
		// Start worker pool
		for i := 0; i < 5; i++ { // 5 workers
			go instance.worker()
		}
	})
	return instance
}

// worker processes tasks from the worker pool
func (m *ClientManager) worker() {
	for task := range m.workerPool {
		task()
	}
}

// GetClient returns a client by ID
func (m *ClientManager) GetClient(id string) (*Client, bool) {
	m.clientsLock.RLock()
	defer m.clientsLock.RUnlock()
	client, exists := m.clients[id]
	return client, exists
}

// AddClient adds a new client
func (m *ClientManager) AddClient(id string, container *sqlstore.Container, whatsmeowClient *whatsmeow.Client) (*Client, error) {
	m.clientsLock.Lock()
	defer m.clientsLock.Unlock()

	if _, exists := m.clients[id]; exists {
		return nil, fmt.Errorf("client with ID %s already exists", id)
	}

	client := &Client{
		ID:             id,
		WhatsmeowClient: whatsmeowClient,
		Container:      container,
		Status:         StatusDisconnected,
		manager:        m,
	}

	// Set up event handler
	whatsmeowClient.AddEventHandler(client.handleWhatsmeowEvent)

	m.clients[id] = client
	m.logger.Printf("Added client with ID %s", id)
	return client, nil
}

// RemoveClient removes a client
func (m *ClientManager) RemoveClient(id string) error {
	m.clientsLock.Lock()
	defer m.clientsLock.Unlock()

	client, exists := m.clients[id]
	if !exists {
		return fmt.Errorf("client with ID %s not found", id)
	}

	// Clean up resources
	if client.WhatsmeowClient.IsConnected() {
		client.WhatsmeowClient.Disconnect()
	}
	if client.Container != nil {
		client.Container.Close()
	}

	delete(m.clients, id)
	m.logger.Printf("Removed client with ID %s", id)
	return nil
}

// ClientExists checks if a client exists
func (m *ClientManager) ClientExists(id string) bool {
	m.clientsLock.RLock()
	defer m.clientsLock.RUnlock()
	_, exists := m.clients[id]
	return exists
}

// GetAllClients returns all clients
func (m *ClientManager) GetAllClients() map[string]*Client {
	m.clientsLock.RLock()
	defer m.clientsLock.RUnlock()
	
	// Create a copy to avoid race conditions
	clientsCopy := make(map[string]*Client, len(m.clients))
	for id, client := range m.clients {
		clientsCopy[id] = client
	}
	
	return clientsCopy
}

// RegisterObserver registers an observer for a specific event type
func (m *ClientManager) RegisterObserver(eventType string, observer Observer) {
	m.observersLock.Lock()
	defer m.observersLock.Unlock()
	
	m.observers[eventType] = append(m.observers[eventType], observer)
	m.logger.Printf("Registered observer for event type %s", eventType)
}

// UnregisterObserver unregisters an observer for a specific event type
func (m *ClientManager) UnregisterObserver(eventType string, observer Observer) {
	m.observersLock.Lock()
	defer m.observersLock.Unlock()
	
	observers, exists := m.observers[eventType]
	if !exists {
		return
	}
	
	// Find and remove the observer
	for i, obs := range observers {
		if obs == observer {
			m.observers[eventType] = append(observers[:i], observers[i+1:]...)
			m.logger.Printf("Unregistered observer for event type %s", eventType)
			break
		}
	}
}

// DispatchEvent dispatches an event to all registered observers
func (m *ClientManager) DispatchEvent(event Event) {
	m.observersLock.RLock()
	observers, exists := m.observers[event.GetType()]
	m.observersLock.RUnlock()
	
	if !exists || len(observers) == 0 {
		return
	}
	
	// Use worker pool to handle event dispatching
	m.workerPool <- func() {
		for _, observer := range observers {
			observer.OnEvent(event)
		}
	}
}

// ForceGC forces garbage collection
func (m *ClientManager) ForceGC() {
	// This is a placeholder for actual GC forcing
	// In a real implementation, you might use runtime.GC()
	m.logger.Println("Forcing garbage collection")
}

// CleanupStaleClients cleans up stale clients
func (m *ClientManager) CleanupStaleClients(maxIdleTime time.Duration) {
	m.clientsLock.Lock()
	defer m.clientsLock.Unlock()
	
	now := time.Now()
	for id, client := range m.clients {
		if client.Status == StatusDisconnected && now.Sub(client.lastActivityTime) > maxIdleTime {
			m.logger.Printf("Cleaning up stale client %s", id)
			
			// Clean up resources
			if client.WhatsmeowClient.IsConnected() {
				client.WhatsmeowClient.Disconnect()
			}
			if client.Container != nil {
				client.Container.Close()
			}
			
			delete(m.clients, id)
		}
	}
}
