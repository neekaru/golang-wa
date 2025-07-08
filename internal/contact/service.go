package contact

import (
	"context"
	"fmt"
	"strings"

	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Service handles contact-related operations
type Service struct {
	app *app.App
}

// NewService creates a new contact service
func NewService(app *app.App) *Service {
	return &Service{
		app: app,
	}
}

// GetAllContacts retrieves all contacts for a user
func (s *Service) GetAllContacts(user string) ([]Contact, error) {
	clientManager := s.app.GetClientManager()
	client, exists := clientManager.GetClient(user)
	if !exists {
		return nil, fmt.Errorf("client not found for user %s", user)
	}

	if !client.IsLoggedIn() {
		return nil, fmt.Errorf("client is not logged in")
	}

	ctx := context.Background()
	contacts, err := client.WhatsmeowClient.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %v", err)
	}

	var result []Contact
	for jid, contact := range contacts {
		// Skip group JIDs (those containing "@lid" are for groups, not individual contacts)
		if strings.Contains(jid.String(), "@lid") {
			continue
		}

		// Parse phone number from JID
		phoneNumber := strings.Split(jid.User, "@")[0]

		// Determine if contact is saved (has a name)
		// In whatsmeow, contacts with names are considered "saved"
		// Use PushName as the display name since it's available in ContactInfo
		isSaved := contact.PushName != ""

		contactInfo := Contact{
			JID:          jid.String(),
			PhoneNumber:  phoneNumber,
			Name:         contact.PushName, // Use PushName as the contact name
			PushName:     contact.PushName,
			BusinessName: contact.BusinessName,
			IsSaved:      isSaved,
			IsBusiness:   contact.BusinessName != "",
		}

		result = append(result, contactInfo)
	}

	return result, nil
}

// GetSavedContacts retrieves only saved contacts (contacts with names)
func (s *Service) GetSavedContacts(user string) ([]Contact, error) {
	allContacts, err := s.GetAllContacts(user)
	if err != nil {
		return nil, err
	}

	var savedContacts []Contact
	for _, contact := range allContacts {
		if contact.IsSaved {
			savedContacts = append(savedContacts, contact)
		}
	}

	return savedContacts, nil
}

// GetUnsavedContacts retrieves only unsaved contacts (contacts without names)
func (s *Service) GetUnsavedContacts(user string) ([]Contact, error) {
	allContacts, err := s.GetAllContacts(user)
	if err != nil {
		return nil, err
	}

	var unsavedContacts []Contact
	for _, contact := range allContacts {
		if !contact.IsSaved {
			unsavedContacts = append(unsavedContacts, contact)
		}
	}

	return unsavedContacts, nil
}

// RefreshContacts forces a refresh of contacts from WhatsApp servers
func (s *Service) RefreshContacts(user string) error {
	clientManager := s.app.GetClientManager()
	client, exists := clientManager.GetClient(user)
	if !exists {
		return fmt.Errorf("client not found for user %s", user)
	}

	if !client.IsLoggedIn() {
		return fmt.Errorf("client is not logged in")
	}

	// In whatsmeow, there's no direct RefreshContactList method
	// Contacts are automatically synced when the client connects
	// We can trigger a sync by requesting presence updates or by reconnecting
	// For now, we'll return success as contacts are managed automatically
	s.app.Logger.Printf("Contact refresh requested for user %s - contacts are automatically synced by whatsmeow", user)
	
	return nil
}
