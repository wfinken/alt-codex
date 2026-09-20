package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/wfinken/alt-codex/internal/paths"
)

// encryptedStore is the FR-02 fallback used when no OS keychain is reachable.
// All secrets are kept AES-256-GCM encrypted in one file; the key lives next
// to it with 0600 permissions, since there is no keychain left to hold it in.
type encryptedStore struct {
	dataPath string
	keyPath  string
}

func newEncryptedStore() (Store, error) {
	dataPath, err := paths.EncryptedSecretsFile()
	if err != nil {
		return nil, err
	}
	keyPath, err := paths.EncryptionKeyFile()
	if err != nil {
		return nil, err
	}
	return &encryptedStore{dataPath: dataPath, keyPath: keyPath}, nil
}

func (s *encryptedStore) Backend() string {
	return "encrypted local file (no OS keychain found)"
}

func (s *encryptedStore) key() ([]byte, error) {
	if b, err := os.ReadFile(s.keyPath); err == nil && len(b) == 32 {
		return b, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(s.keyPath, key, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

func (s *encryptedStore) load() (map[string]string, error) {
	out := map[string]string{}
	raw, err := os.ReadFile(s.dataPath)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return out, nil
	}

	key, err := s.key()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, errors.New("corrupt secret store")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret store (wrong or missing key): %w", err)
	}
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *encryptedStore) save(m map[string]string) error {
	key, err := s.key()
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	plain, err := json.Marshal(m)
	if err != nil {
		return err
	}
	ciphertext := gcm.Seal(nonce, nonce, plain, nil)
	tmp := s.dataPath + ".tmp"
	if err := os.WriteFile(tmp, ciphertext, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.dataPath)
}

func (s *encryptedStore) Set(profileName, value string) error {
	m, err := s.load()
	if err != nil {
		return err
	}
	m[profileName] = value
	return s.save(m)
}

func (s *encryptedStore) Get(profileName string) (string, error) {
	m, err := s.load()
	if err != nil {
		return "", err
	}
	v, ok := m[profileName]
	if !ok {
		return "", fmt.Errorf("no secret stored for %q", profileName)
	}
	return v, nil
}

func (s *encryptedStore) Delete(profileName string) error {
	m, err := s.load()
	if err != nil {
		return err
	}
	delete(m, profileName)
	return s.save(m)
}
