// Package backup implements the roadmap's "encrypted import/export" feature:
// a single portable file holding every profile's metadata and secret, so it
// can be moved to another machine or kept somewhere safe as a restore point.
//
// Unlike internal/secret's encrypted fallback — whose key lives unprotected
// next to the ciphertext, since it only has to resist someone without disk
// access — an export file is meant to travel, so its key is derived from a
// passphrase the caller supplies, via scrypt, and never stored anywhere.
package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/scrypt"

	"github.com/wfinken/alt-codex/internal/profile"
)

// FormatVersion identifies the archive's on-disk shape, bumped only if it
// ever needs to change incompatibly.
const FormatVersion = 1

// magic tags a decrypted-looking-but-wrong file (or a plain passphrase typo
// against random bytes) with a clear error instead of a cryptic JSON one.
var magic = [8]byte{'A', 'L', 'T', 'C', 'D', 'X', 'B', '1'}

const (
	saltSize = 16
	keySize  = 32
	scryptN  = 1 << 15
	scryptR  = 8
	scryptP  = 1
)

// Archive is the plaintext payload, JSON-marshaled and then encrypted.
type Archive struct {
	Version    int               `json:"version"`
	ExportedAt time.Time         `json:"exported_at"`
	Profiles   []profile.Profile `json:"profiles"`
	// Secrets maps a profile name to its credential value, exactly as
	// internal/secret.Store would return it for that name.
	Secrets map[string]string `json:"secrets"`
}

func deriveKey(passphrase string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, keySize)
}

// Encrypt serializes and encrypts a into a self-contained archive file: an
// 8-byte magic header, a random salt, a random GCM nonce, then ciphertext.
// The passphrase is never stored; it must be supplied again to Decrypt.
func Encrypt(a Archive, passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase must not be empty")
	}
	plain, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key, err := deriveKey(passphrase, salt)
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
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	out := make([]byte, 0, len(magic)+len(salt)+len(nonce)+len(plain)+gcm.Overhead())
	out = append(out, magic[:]...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = gcm.Seal(out, nonce, plain, nil)
	return out, nil
}

// Decrypt reverses Encrypt. A wrong passphrase and a corrupt file are
// reported the same way (GCM authentication fails for both), since telling
// them apart isn't possible and wouldn't help the caller either way.
func Decrypt(data []byte, passphrase string) (Archive, error) {
	if passphrase == "" {
		return Archive{}, errors.New("passphrase must not be empty")
	}
	if len(data) < len(magic) || [8]byte(data[:len(magic)]) != magic {
		return Archive{}, errors.New("not an alt-codex export file")
	}
	data = data[len(magic):]

	if len(data) < saltSize {
		return Archive{}, errors.New("corrupt export file")
	}
	salt, data := data[:saltSize], data[saltSize:]
	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return Archive{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Archive{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Archive{}, err
	}
	if len(data) < gcm.NonceSize() {
		return Archive{}, errors.New("corrupt export file")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return Archive{}, fmt.Errorf("decrypt export file (wrong passphrase or corrupt file): %w", err)
	}

	var a Archive
	if err := json.Unmarshal(plain, &a); err != nil {
		return Archive{}, err
	}
	return a, nil
}
