// Package codexconfig applies an alt-codex profile's credential to the local
// Codex CLI's own auth file, which is what actually makes a "switch" take
// effect for the Codex CLI itself (FR-04).
package codexconfig

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/wfinken/alt-codex/internal/profile"
)

// TargetPath returns the Codex CLI auth file alt-codex writes to on switch.
// It honors $ALT_CODEX_TARGET (used by tests and by anyone pointing at a
// non-default Codex CLI install) and otherwise defaults to ~/.codex/auth.json.
func TargetPath() (string, error) {
	if p := os.Getenv("ALT_CODEX_TARGET"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "auth.json"), nil
}

// Apply writes secret to the Codex CLI auth file in the shape appropriate
// for typ, activating it for the Codex CLI immediately.
func Apply(typ profile.CredentialType, secret string) error {
	target, err := TargetPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}

	var payload []byte
	switch typ {
	case profile.TypeRawJSON:
		// Already a full auth.json document; re-marshal to normalize formatting.
		var v any
		if err := json.Unmarshal([]byte(secret), &v); err != nil {
			return err
		}
		payload, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
	default: // profile.TypeAPIKey
		payload, err = json.MarshalIndent(map[string]string{"OPENAI_API_KEY": secret}, "", "  ")
		if err != nil {
			return err
		}
	}

	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}
