package secret

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

type keyringStore struct{}

// probeAccount is a canary entry used only to confirm the OS keychain is
// actually reachable (e.g. Secret Service running on Linux) before we
// commit to using it for real profiles. The probe reads rather than writes:
// a lookup for an item that doesn't exist returns ErrNotFound instantly with
// no OS permission prompt, whereas writing (even a throwaway value) would
// trigger a real "app wants to access your keychain" dialog on every
// startup — unacceptable for NFR-01 and for user trust.
const probeAccount = "__alt-codex-probe__"

func newKeyringStore() (Store, error) {
	if _, err := keyring.Get(service, probeAccount); err != nil && err != keyring.ErrNotFound {
		return nil, fmt.Errorf("os keychain unavailable: %w", err)
	}
	return keyringStore{}, nil
}

func (keyringStore) Backend() string { return "OS keychain" }

func (keyringStore) Set(profileName, value string) error {
	return keyring.Set(service, profileName, value)
}

func (keyringStore) Get(profileName string) (string, error) {
	v, err := keyring.Get(service, profileName)
	if err != nil {
		return "", fmt.Errorf("read secret for %q: %w", profileName, err)
	}
	return v, nil
}

func (keyringStore) Delete(profileName string) error {
	if err := keyring.Delete(service, profileName); err != nil && err != keyring.ErrNotFound {
		return err
	}
	return nil
}
