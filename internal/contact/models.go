package contact

// UserRequest represents a request with user authentication
type UserRequest struct {
	User string `json:"user" binding:"required"`
}

// Contact represents a WhatsApp contact
type Contact struct {
	JID          string `json:"jid"`           // WhatsApp JID (e.g., "1234567890@s.whatsapp.net")
	PhoneNumber  string `json:"phone_number"`  // Phone number without country code formatting
	Name         string `json:"name"`          // Contact name (empty if not saved)
	PushName     string `json:"push_name"`     // Name set by the contact themselves
	BusinessName string `json:"business_name"` // Business name if it's a business contact
	IsSaved      bool   `json:"is_saved"`      // Whether this contact is saved in the phone
	IsBusiness   bool   `json:"is_business"`   // Whether this is a business contact
}

// ContactsResponse represents the response for contact listing
type ContactsResponse struct {
	Contacts []Contact `json:"contacts"`
	Total    int       `json:"total"`
	User     string    `json:"user"`
}