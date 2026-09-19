package utils

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// CheckWhatsAppNumber asks WhatsApp for the canonical identity of a phone number.
func CheckWhatsAppNumber(ctx context.Context, client *whatsmeow.Client, phoneNumber string) (types.IsOnWhatsAppResponse, error) {
	phoneNumber = strings.TrimPrefix(strings.TrimSpace(phoneNumber), "+")
	if phoneNumber == "" || strings.IndexFunc(phoneNumber, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return types.IsOnWhatsAppResponse{}, fmt.Errorf("phone number must contain only digits with an optional leading +")
	}

	users, err := client.IsOnWhatsApp(ctx, []string{"+" + phoneNumber})
	if err != nil {
		return types.IsOnWhatsAppResponse{}, fmt.Errorf("failed to check WhatsApp recipient: %w", err)
	} else if len(users) == 0 {
		return types.IsOnWhatsAppResponse{}, fmt.Errorf("WhatsApp returned no result for phone number")
	}
	return users[0], nil
}

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

	user, err := CheckWhatsAppNumber(ctx, client, phoneNumber)
	if err != nil {
		return types.EmptyJID, err
	}
	if !user.IsIn {
		return types.EmptyJID, fmt.Errorf("phone number is not registered on WhatsApp")
	} else if user.JID.IsEmpty() || user.JID.Server != types.HiddenUserServer {
		return types.EmptyJID, fmt.Errorf("WhatsApp account found but LID is unavailable")
	}
	return user.JID, nil
}
