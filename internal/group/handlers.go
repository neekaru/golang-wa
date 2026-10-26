package group

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Handlers contains HTTP handlers for group management.
type Handlers struct {
	app     *app.App
	service *Service
}

// NewHandlers creates group handlers.
func NewHandlers(app *app.App) *Handlers {
	return &Handlers{app: app, service: NewService(app)}
}

// ListGroupsHandler handles POST /groups/list.
func (h *Handlers) ListGroupsHandler(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request. Required: {\"user\": \"username\"}"})
		return
	}
	if strings.TrimSpace(req.User) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User field is required"})
		return
	}

	groups, err := h.service.ListGroups(c.Request.Context(), req.User)
	if err != nil {
		h.app.Logger.Printf("List groups error for user %s: %v", req.User, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get groups",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GroupsResponse{
		Groups: groups,
		Total:  len(groups),
		User:   req.User,
	})
}
