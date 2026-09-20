// Package secret stores and retrieves profile credentials, preferring the
// OS-native keychain and transparently falling back to an encrypted local
// file when no keychain is available (FR-02, NFR-03).
package secret

const service = "alt-codex"

// Store saves, loads, and deletes a single secret per profile name.
type Store interface {
	// Backend reports a short, human-readable name for status/debug display.
	Backend() string
	Set(profileName, value string) error
	Get(profileName string) (string, error)
	Delete(profileName string) error
}

// New picks the OS keychain when it is usable on this machine, falling back
// to an encrypted file store otherwise.
func New() (Store, error) {
	if ks, err := newKeyringStore(); err == nil {
		return ks, nil
	}
	return newEncryptedStore()
}
