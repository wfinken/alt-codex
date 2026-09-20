package secret

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

type keyringStore struct{}

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
