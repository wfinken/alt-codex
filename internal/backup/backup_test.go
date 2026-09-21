package backup

import (
	"testing"
	"time"

	"github.com/wfinken/alt-codex/internal/profile"
)

func testArchive() Archive {
	return Archive{
		Version:    FormatVersion,
		ExportedAt: time.Now().UTC(),
		Profiles: []profile.Profile{
			{Name: "work", Type: profile.TypeAPIKey, CreatedAt: time.Now().UTC()},
		},
		Secrets: map[string]string{"work": "sk-super-secret"},
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	a := testArchive()

	data, err := Encrypt(a, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := Decrypt(data, "correct horse battery staple")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Name != "work" {
		t.Fatalf("Profiles = %+v, want one profile named work", got.Profiles)
	}
	if got.Secrets["work"] != "sk-super-secret" {
		t.Fatalf("Secrets[work] = %q, want %q", got.Secrets["work"], "sk-super-secret")
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	data, err := Encrypt(testArchive(), "right-passphrase")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := Decrypt(data, "wrong-passphrase"); err == nil {
		t.Fatal("expected error decrypting with wrong passphrase")
	}
}

func TestDecryptCorruptFile(t *testing.T) {
	if _, err := Decrypt([]byte("not an export file"), "whatever"); err == nil {
		t.Fatal("expected error decrypting a non-archive file")
	}
	if _, err := Decrypt(magic[:], "whatever"); err == nil {
		t.Fatal("expected error decrypting a truncated archive")
	}
}

func TestEncryptEmptyPassphrase(t *testing.T) {
	if _, err := Encrypt(testArchive(), ""); err == nil {
		t.Fatal("expected error encrypting with an empty passphrase")
	}
}

func TestDecryptEmptyPassphrase(t *testing.T) {
	data, err := Encrypt(testArchive(), "some-passphrase")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := Decrypt(data, ""); err == nil {
		t.Fatal("expected error decrypting with an empty passphrase")
	}
}
