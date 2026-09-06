package web

import (
	"errors"
	"strings"

	"go.mau.fi/whatsmeow/types"
)

var errBadRecipient = errors.New("invalid recipient")

// normaliseRecipient turns form input into a JID string the scheduler can send to.
// Accepts a bare phone number (digits, with optional +, spaces, dashes) or a full JID
// on one of the servers we send to. No network lookup: a wrong number surfaces as
// failed at send time.
func normaliseRecipient(input string) (string, error) {
	input = strings.TrimSpace(input)
	if !strings.Contains(input, "@") {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, input)
		if digits == "" {
			return "", errBadRecipient
		}
		return types.NewJID(digits, types.DefaultUserServer).String(), nil
	}

	jid, err := types.ParseJID(input)
	if err != nil {
		return "", errBadRecipient
	}
	switch jid.Server {
	case types.DefaultUserServer:
		if jid.User == "" || strings.Trim(jid.User, "0123456789") != "" {
			return "", errBadRecipient
		}
	case types.GroupServer:
	default:
		return "", errBadRecipient
	}
	return jid.String(), nil
}
