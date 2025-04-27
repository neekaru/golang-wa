package client

// Observer is the interface for event observers
type Observer interface {
	OnEvent(event Event)
}

// ObserverFunc is a function that implements the Observer interface
type ObserverFunc func(event Event)

// OnEvent calls the observer function
func (f ObserverFunc) OnEvent(event Event) {
	f(event)
}

// FilteredObserver is an observer that only receives events of a specific type
type FilteredObserver struct {
	EventType string
	Observer  Observer
}

// NewFilteredObserver creates a new filtered observer
func NewFilteredObserver(eventType string, observer Observer) *FilteredObserver {
	return &FilteredObserver{
		EventType: eventType,
		Observer:  observer,
	}
}

// OnEvent calls the underlying observer if the event type matches
func (f *FilteredObserver) OnEvent(event Event) {
	if event.GetType() == f.EventType {
		f.Observer.OnEvent(event)
	}
}

// ClientFilteredObserver is an observer that only receives events for a specific client
type ClientFilteredObserver struct {
	ClientID string
	Observer Observer
}

// NewClientFilteredObserver creates a new client-filtered observer
func NewClientFilteredObserver(clientID string, observer Observer) *ClientFilteredObserver {
	return &ClientFilteredObserver{
		ClientID: clientID,
		Observer: observer,
	}
}

// OnEvent calls the underlying observer if the client ID matches
func (f *ClientFilteredObserver) OnEvent(event Event) {
	if event.GetClientID() == f.ClientID {
		f.Observer.OnEvent(event)
	}
}
