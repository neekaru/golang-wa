package utils

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ResolveRecipient returns the canonical WhatsApp JID for a phone number.
func ResolveRecipient(ctx context.Context, client *whatsmeow.Client, phoneNumber string) (types.JID, error) {
	phoneNumber = strings.TrimPrefix(phoneNumber, "+")
	phoneJID := types.NewJID(phoneNumber, types.DefaultUserServer)

	lid, err := client.Store.LIDs.GetLIDForPN(ctx, phoneJID)
	if err != nil {
		return types.EmptyJID, fmt.Errorf("failed to resolve WhatsApp recipient: %w", err)
	} else if !lid.IsEmpty() {
		return lid, nil
	}

	users, err := client.IsOnWhatsApp(ctx, []string{"+" + phoneNumber})
	if err != nil {
		return types.EmptyJID, fmt.Errorf("failed to check WhatsApp recipient: %w", err)
	}
	return recipientFromLookup(users)
}

func recipientFromLookup(users []types.IsOnWhatsAppResponse) (types.JID, error) {
	if len(users) == 0 || !users[0].IsIn {
		return types.EmptyJID, fmt.Errorf("phone number is not registered on WhatsApp")
	} else if users[0].JID.IsEmpty() || users[0].JID.Server != types.HiddenUserServer {
		return types.EmptyJID, fmt.Errorf("WhatsApp account found but LID is unavailable")
	}
	return users[0].JID, nil
}
