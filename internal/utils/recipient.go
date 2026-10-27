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

// ParseRecipient accepts a phone number or a modern/legacy WhatsApp group JID.
func ParseRecipient(recipient string) (types.JID, error) {
	if strings.HasSuffix(recipient, "@"+types.GroupServer) {
		user := strings.TrimSuffix(recipient, "@"+types.GroupServer)
		first, second, legacy := strings.Cut(user, "-")
		if isDigits(first) && (!legacy || isDigits(second)) {
			return types.NewJID(user, types.GroupServer), nil
		}
	} else {
		phone := strings.TrimPrefix(recipient, "+")
		if isDigits(phone) {
			return types.NewJID(phone, types.DefaultUserServer), nil
		}
	}
	return types.EmptyJID, fmt.Errorf("recipient is invalid, must be a phone number or a group JID ending in @g.us")
}

func isDigits(value string) bool {
	return value != "" && strings.IndexFunc(value, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

// ResolveRecipient returns a group JID directly or resolves a phone number to its LID.
func ResolveRecipient(ctx context.Context, client *whatsmeow.Client, phoneNumber string) (types.JID, error) {
	phoneJID, err := ParseRecipient(phoneNumber)
	if err != nil {
		return types.EmptyJID, err
	}
	if phoneJID.Server == types.GroupServer {
		return phoneJID, nil
	}

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
