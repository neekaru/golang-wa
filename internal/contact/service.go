package contact

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"

	"github.com/neekaru/whatsappgo-bot/internal/app"
	"github.com/neekaru/whatsappgo-bot/internal/utils"
)

func contactDisplayName(contactName, pushName, businessName string) string {
	switch {
	case contactName != "":
		return contactName
	case pushName != "":
		return pushName
	case businessName != "":
		return businessName
	default:
		return ""
	}
}

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

// CheckWhatsAppNumber checks whether a phone number is registered on WhatsApp.
func (s *Service) CheckWhatsAppNumber(user, phoneNumber string) (CheckNumberResponse, error) {
	client, exists := s.app.GetClientManager().GetClient(user)
	if !exists {
		return CheckNumberResponse{}, fmt.Errorf("client not found for user %s", user)
	} else if !client.IsLoggedIn() {
		return CheckNumberResponse{}, fmt.Errorf("client is not logged in")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := utils.CheckWhatsAppNumber(ctx, client.WhatsmeowClient, phoneNumber)
	if err != nil {
		return CheckNumberResponse{}, err
	}

	response := CheckNumberResponse{
		User:        user,
		PhoneNumber: strings.TrimPrefix(strings.TrimSpace(phoneNumber), "+"),
		Registered:  result.IsIn,
	}
	if result.IsIn && !result.JID.IsEmpty() {
		response.JID = result.JID.String()
	}
	return response, nil
}

// GetAllContacts retrieves all contacts for a user
func (s *Service) GetAllContacts(user string) ([]Contact, error) {
	clientManager := s.app.GetClientManager()
	client, exists := clientManager.GetClient(user)
	if !exists {
		return nil, fmt.Errorf("client not found for user %s", user)
	}

	ctx := context.Background()
	contacts, err := client.WhatsmeowClient.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %v", err)
	}

	var result []Contact
	for jid, contact := range contacts {
		// Skip groups and broadcasts
		if jid.Server == types.GroupServer || jid.Server == types.BroadcastServer {
			continue
		}

		// Resolve phone number: if JID is LID, try to lookup phone number from store mapping
		phoneNumber := strings.Split(jid.User, "@")[0]
		if jid.Server == types.HiddenUserServer && client.WhatsmeowClient.Store.LIDs != nil {
			if pn, err := client.WhatsmeowClient.Store.LIDs.GetPNForLID(ctx, jid); err == nil && !pn.IsEmpty() {
				phoneNumber = pn.User
			}
		}

		displayName := contactDisplayName(contact.FullName, contact.PushName, contact.BusinessName)
		isSaved := displayName != ""

		contactInfo := Contact{
			JID:          jid.String(),
			PhoneNumber:  phoneNumber,
			Name:         displayName,
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
