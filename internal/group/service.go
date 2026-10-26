package group

import (
	"context"
	"fmt"
	"time"

	"github.com/neekaru/whatsappgo-bot/internal/app"
)

// Service handles group-related operations.
type Service struct {
	app *app.App
}

// NewService creates a group service.
func NewService(app *app.App) *Service {
	return &Service{app: app}
}

// ListGroups retrieves the groups the user's WhatsApp account has joined.
func (s *Service) ListGroups(ctx context.Context, user string) ([]Group, error) {
	client, exists := s.app.GetClientManager().GetClient(user)
	if !exists {
		return nil, fmt.Errorf("client not found for user %s", user)
	}
	if !client.IsLoggedIn() {
		return nil, fmt.Errorf("client is not logged in")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	groups, err := client.WhatsmeowClient.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get joined groups: %w", err)
	}

	result := make([]Group, len(groups))
	for i, info := range groups {
		result[i] = Group{JID: info.JID.String(), Name: info.Name}
	}
	return result, nil
}
