package client

import (
	"log"
	"os"
)

// LoggingObserver is a simple observer that logs events
type LoggingObserver struct {
	logger *log.Logger
}

// NewLoggingObserver creates a new logging observer
func NewLoggingObserver(logger *log.Logger) *LoggingObserver {
	return &LoggingObserver{
		logger: logger,
	}
}

// OnEvent logs the event
func (o *LoggingObserver) OnEvent(event Event) {
	o.logger.Printf("Event: type=%s, clientID=%s", event.GetType(), event.GetClientID())
}

// StatusChangeObserver is an observer that handles status change events
type StatusChangeObserver struct {
	onStatusChange func(clientID string, status ClientStatus)
}

// NewStatusChangeObserver creates a new status change observer
func NewStatusChangeObserver(callback func(clientID string, status ClientStatus)) *StatusChangeObserver {
	return &StatusChangeObserver{
		onStatusChange: callback,
	}
}

// OnEvent handles the event
func (o *StatusChangeObserver) OnEvent(event Event) {
	if event.GetType() == EventTypeStatus {
		if statusEvent, ok := event.(*StatusEvent); ok {
			o.onStatusChange(statusEvent.ClientID, statusEvent.Status)
		}
	}
}

// QRCodeObserver is an observer that handles QR code events
type QRCodeObserver struct {
	onQRCode func(clientID string, qrEvent interface{})
}

// NewQRCodeObserver creates a new QR code observer
func NewQRCodeObserver(callback func(clientID string, qrEvent interface{})) *QRCodeObserver {
	return &QRCodeObserver{
		onQRCode: callback,
	}
}

// OnEvent handles the event
func (o *QRCodeObserver) OnEvent(event Event) {
	if event.GetType() == EventTypeQR {
		if qrEvent, ok := event.(*QREvent); ok {
			o.onQRCode(qrEvent.ClientID, qrEvent.QREvent)
		}
	}
}

// Example of how to use the observers
func ExampleRegisterObservers(clientManager *ClientManager, logger *log.Logger) {
	// Register a logging observer for all events
	clientManager.RegisterObserver(EventTypeStatus, NewLoggingObserver(logger))
	clientManager.RegisterObserver(EventTypeQR, NewLoggingObserver(logger))
	clientManager.RegisterObserver(EventTypeError, NewLoggingObserver(logger))

	// Register a status change observer
	clientManager.RegisterObserver(EventTypeStatus, NewStatusChangeObserver(func(clientID string, status ClientStatus) {
		logger.Printf("Client %s status changed to %s", clientID, status.String())

		// Example of taking action based on status change
		if status == StatusLoggedIn {
			logger.Printf("Client %s is now logged in, can start sending messages", clientID)
		} else if status == StatusDisconnected {
			logger.Printf("Client %s is now disconnected, should reconnect", clientID)
		}
	}))

	// Register a QR code observer
	clientManager.RegisterObserver(EventTypeQR, NewQRCodeObserver(func(clientID string, qrEvent interface{}) {
		logger.Printf("Client %s received QR code, should display it to user", clientID)

		// Example of how to handle QR code
		// qrCode := qrEvent.(*events.QR)
		// displayQRCode(qrCode)
	}))
}

// Example of how to create a client-specific observer
func ExampleClientSpecificObserver(clientManager *ClientManager, clientID string, logger *log.Logger) {
	// Create a client-specific observer
	clientObserver := NewClientFilteredObserver(clientID, ObserverFunc(func(event Event) {
		logger.Printf("Client %s event: %s", clientID, event.GetType())
	}))

	// Register the observer for all event types
	clientManager.RegisterObserver(EventTypeStatus, clientObserver)
	clientManager.RegisterObserver(EventTypeQR, clientObserver)
	clientManager.RegisterObserver(EventTypeError, clientObserver)
}

// Example of how to use the ClientManager
func ExampleUseClientManager() {
	// Get the singleton instance
	clientManager := GetInstance()

	// Create a logger
	logger := log.New(os.Stdout, "ClientManager: ", log.LstdFlags)

	// Register observers
	ExampleRegisterObservers(clientManager, logger)

	// Create a client
	// container, _ := sqlstore.New("sqlite3", "file:data/user1.db?_foreign_keys=on", nil)
	// deviceStore, _ := container.GetFirstDevice()
	// whatsmeowClient := whatsmeow.NewClient(deviceStore, nil)
	// client, _ := clientManager.AddClient("user1", container, whatsmeowClient)

	// Connect the client
	// client.Connect()

	// Register a client-specific observer
	// ExampleClientSpecificObserver(clientManager, "user1", logger)

	// Get all clients
	clients := clientManager.GetAllClients()
	for id, client := range clients {
		logger.Printf("Client %s status: %s", id, client.Status.String())
	}

	// Remove a client
	// clientManager.RemoveClient("user1")
}
