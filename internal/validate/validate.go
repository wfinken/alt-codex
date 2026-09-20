// Package validate implements the credential sanity checks required before
// a profile is saved (FR-07). It catches obviously malformed input; it does
// not call out to any network service to confirm a token is still live.
package validate

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/wfinken/alt-codex/internal/profile"
)

var (
	// ErrEmpty is returned when a required field was left blank.
	ErrEmpty = errors.New("value cannot be empty")
	// ErrTooShort flags tokens implausibly short to be real credentials.
	ErrTooShort = errors.New("token looks too short to be a real credential")
	// ErrWhitespace flags copy/paste mistakes (leading/trailing whitespace or newlines).
	ErrWhitespace = errors.New("token contains leading/trailing whitespace")
	// ErrInvalidJSON is returned for raw_json credentials that don't parse.
	ErrInvalidJSON = errors.New("not valid JSON")
)

const minAPIKeyLength = 12

// ProfileName checks a proposed profile name is non-empty and free of
// characters that would be awkward in a keychain account name or file path.
func ProfileName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrEmpty
	}
	if name != strings.TrimSpace(name) {
		return ErrWhitespace
	}
	for _, r := range name {
		if r == '/' || r == '\\' || r < ' ' {
			return errors.New("name cannot contain slashes or control characters")
		}
	}
	return nil
}

// Credential checks a secret value before it is persisted, according to the
// profile's declared credential type.
func Credential(typ profile.CredentialType, value string) error {
	if value == "" {
		return ErrEmpty
	}
	if strings.TrimSpace(value) != value {
		return ErrWhitespace
	}

	switch typ {
	case profile.TypeRawJSON:
		var v any
		if err := json.Unmarshal([]byte(value), &v); err != nil {
			return ErrInvalidJSON
		}
		return nil
	case profile.TypeAPIKey:
		fallthrough
	default:
		if len(value) < minAPIKeyLength {
			return ErrTooShort
		}
		return nil
	}
}

// ExpiresAt parses an optional YYYY-MM-DD expiry date. An empty string is
// valid and means "no known expiry".
func ExpiresAt(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, errors.New("use YYYY-MM-DD format")
	}
	return &t, nil
}
