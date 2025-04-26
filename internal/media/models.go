package media

// SendMediaRequest represents a request to send media
type SendMediaRequest struct {
	User        string `json:"user"`
	PhoneNumber string `json:"phone_number"`
	Media       string `json:"media"`
	URL         string `json:"url"`
	Caption     string `json:"caption"`
	FileName    string `json:"file_name"` // Optional filename parameter
}
