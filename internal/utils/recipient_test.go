package utils

import (
	"context"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestParseRecipient(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: "6281234567890", want: "6281234567890@s.whatsapp.net"},
		{input: "+6281234567890", want: "6281234567890@s.whatsapp.net"},
		{input: "120363123456789012@g.us", want: "120363123456789012@g.us"},
		{input: "6281234567890-1234567890@g.us", want: "6281234567890-1234567890@g.us"},
		{input: ""},
		{input: "+"},
		{input: " 6281234567890"},
		{input: "628 1234567890"},
		{input: "１２３"},
		{input: "@g.us"},
		{input: "team@g.us"},
		{input: "+123@g.us"},
		{input: "123-@g.us"},
		{input: "-456@g.us"},
		{input: "123--456@g.us"},
		{input: "123-456-789@g.us"},
		{input: "123:4@g.us"},
		{input: "123@g.us\n"},
		{input: "123@s.whatsapp.net"},
		{input: "123@lid"},
		{input: "123@g.us@evil"},
		{input: "status@broadcast"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			jid, err := ParseRecipient(tc.input)
			if tc.want == "" {
				if err == nil || jid != types.EmptyJID {
					t.Fatalf("invalid recipient accepted: jid = %v, err = %v", jid, err)
				}
				return
			}
			if err != nil || jid.String() != tc.want {
				t.Fatalf("ParseRecipient(%q) = %v, %v; want %s", tc.input, jid, err, tc.want)
			}
		})
	}
}

func TestResolveRecipientGroupBypassesPhoneLookup(t *testing.T) {
	const group = "120363123456789012@g.us"
	// A nil client ensures groups never access the phone/LID store or network lookup.
	jid, err := ResolveRecipient(context.Background(), nil, group)
	if err != nil || jid.String() != group || jid.Server != types.GroupServer {
		t.Fatalf("ResolveRecipient(group) = %v, %v; want %s", jid, err, group)
	}
}
