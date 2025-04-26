package session

// AddSessionRequest represents a request to add a new session
type AddSessionRequest struct {
	User string `json:"user"`
}

// StatusResponse represents a session status response
type StatusResponse struct {
	LoggedIn   bool   `json:"logged_in"`
	Connected  bool   `json:"connected"`
	User       string `json:"user"`
	NeedsQR    bool   `json:"needs_qr"`
	Timestamp  string `json:"timestamp"`
}

// LogoutRequest represents a request to logout a session
type LogoutRequest struct {
	User string `json:"user"`
}
