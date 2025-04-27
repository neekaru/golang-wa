package client

import (
	"go.mau.fi/whatsmeow/types/events"
)

// Event is the interface for all events
type Event interface {
	GetType() string
	GetClientID() string
	GetData() interface{}
}

// BaseEvent is the base implementation of Event
type BaseEvent struct {
	Type     string
	ClientID string
	Data     interface{}
}

// GetType returns the event type
func (e *BaseEvent) GetType() string {
	return e.Type
}

// GetClientID returns the client ID
func (e *BaseEvent) GetClientID() string {
	return e.ClientID
}

// GetData returns the event data
func (e *BaseEvent) GetData() interface{} {
	return e.Data
}

// Event types
const (
	EventTypeStatus = "status"
	EventTypeQR     = "qr"
	EventTypeError  = "error"
	EventTypeRaw    = "raw"
)

// StatusEvent represents a client status change event
type StatusEvent struct {
	BaseEvent
	Status ClientStatus
}

// NewStatusEvent creates a new status event
func NewStatusEvent(clientID string, status ClientStatus) *StatusEvent {
	return &StatusEvent{
		BaseEvent: BaseEvent{
			Type:     EventTypeStatus,
			ClientID: clientID,
			Data:     status,
		},
		Status: status,
	}
}

// QREvent represents a QR code event
type QREvent struct {
	BaseEvent
	QREvent *events.QR
}

// NewQREvent creates a new QR event
func NewQREvent(clientID string, qrEvent *events.QR) *QREvent {
	return &QREvent{
		BaseEvent: BaseEvent{
			Type:     EventTypeQR,
			ClientID: clientID,
			Data:     qrEvent,
		},
		QREvent: qrEvent,
	}
}

// ErrorEvent represents an error event
type ErrorEvent struct {
	BaseEvent
	Error string
}

// NewErrorEvent creates a new error event
func NewErrorEvent(clientID string, errorMsg string) *ErrorEvent {
	return &ErrorEvent{
		BaseEvent: BaseEvent{
			Type:     EventTypeError,
			ClientID: clientID,
			Data:     errorMsg,
		},
		Error: errorMsg,
	}
}

// RawEvent represents a raw whatsmeow event
type RawEvent struct {
	BaseEvent
}

// NewRawEvent creates a new raw event
func NewRawEvent(clientID string, rawEvent interface{}) *RawEvent {
	return &RawEvent{
		BaseEvent: BaseEvent{
			Type:     EventTypeRaw,
			ClientID: clientID,
			Data:     rawEvent,
		},
	}
}
