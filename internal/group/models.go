package group

// UserRequest identifies the WhatsApp session to query.
type UserRequest struct {
	User string `json:"user" binding:"required"`
}

// Group contains the group address and display name.
type Group struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

// GroupsResponse represents the joined groups for a user.
type GroupsResponse struct {
	Groups []Group `json:"groups"`
	Total  int     `json:"total"`
	User   string  `json:"user"`
}
