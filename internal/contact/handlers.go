package contact

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Handlers contains HTTP handlers for contact management
type Handlers struct {
	app     *app.App
	service *Service
}

// NewHandlers creates a new contact handlers instance
func NewHandlers(app *app.App) *Handlers {
	return &Handlers{
		app:     app,
		service: NewService(app),
	}
}

// GetAllContactsHandler handles GET /contact - returns all contacts
func (h *Handlers) GetAllContactsHandler(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request. Required: {\"user\": \"username\"}"})
		return
	}

	if req.User == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User field is required"})
		return
	}

	contacts, err := h.service.GetAllContacts(req.User)
	if err != nil {
		h.app.Logger.Printf("Get all contacts error for user %s: %v", req.User, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get contacts",
			"details": err.Error(),
		})
		return
	}

	response := ContactsResponse{
		Contacts: contacts,
		Total:    len(contacts),
		User:     req.User,
	}

	c.JSON(http.StatusOK, response)
}

// GetSavedContactsHandler handles GET /contact/saved - returns only saved contacts
func (h *Handlers) GetSavedContactsHandler(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request. Required: {\"user\": \"username\"}"})
		return
	}

	if req.User == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User field is required"})
		return
	}

	contacts, err := h.service.GetSavedContacts(req.User)
	if err != nil {
		h.app.Logger.Printf("Get saved contacts error for user %s: %v", req.User, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get saved contacts",
			"details": err.Error(),
		})
		return
	}

	response := ContactsResponse{
		Contacts: contacts,
		Total:    len(contacts),
		User:     req.User,
	}

	c.JSON(http.StatusOK, response)
}

// GetUnsavedContactsHandler handles GET /contact/unsaved - returns only unsaved contacts
func (h *Handlers) GetUnsavedContactsHandler(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request. Required: {\"user\": \"username\"}"})
		return
	}

	if req.User == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User field is required"})
		return
	}

	contacts, err := h.service.GetUnsavedContacts(req.User)
	if err != nil {
		h.app.Logger.Printf("Get unsaved contacts error for user %s: %v", req.User, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get unsaved contacts",
			"details": err.Error(),
		})
		return
	}

	response := ContactsResponse{
		Contacts: contacts,
		Total:    len(contacts),
		User:     req.User,
	}

	c.JSON(http.StatusOK, response)
}

// RefreshContactsHandler handles POST /contact/refresh - refreshes contact list from WhatsApp
func (h *Handlers) RefreshContactsHandler(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request. Required: {\"user\": \"username\"}"})
		return
	}

	if req.User == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User field is required"})
		return
	}

	err := h.service.RefreshContacts(req.User)
	if err != nil {
		h.app.Logger.Printf("Refresh contacts error for user %s: %v", req.User, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refresh contacts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":  "Contacts refreshed successfully",
		"user": req.User,
	})
}