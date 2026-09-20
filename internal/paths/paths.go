// Package paths resolves the on-disk locations alt-codex reads and writes,
// matching the layout described in the project's PRD (~/.config/alt-codex/*)
// on every platform for consistency and easy support/debugging.
package paths

import (
	"os"
	"path/filepath"
)

const dirName = "alt-codex"

// ConfigDir returns ~/.config/alt-codex, creating it if it does not exist.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", dirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// ProfilesFile returns the path to profiles.json.
func ProfilesFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profiles.json"), nil
}

// EncryptedSecretsFile returns the path to the fallback encrypted secret store.
func EncryptedSecretsFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.enc"), nil
}

// EncryptionKeyFile returns the path to the local key used to encrypt
// EncryptedSecretsFile when no OS keychain is available to hold it instead.
func EncryptionKeyFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".secret.key"), nil
}
