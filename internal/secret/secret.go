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

// New returns a Store that prefers the OS keychain, falling back to an
// encrypted local file — either because no keychain is reachable at all, or
// because a specific secret doesn't fit in one (FR-02, NFR-03).
func New() (Store, error) {
	fallback, err := newEncryptedStore()
	if err != nil {
		return nil, err
	}
	return &hybridStore{primary: keyringStore{}, fallback: fallback}, nil
}
