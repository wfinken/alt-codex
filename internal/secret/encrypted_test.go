package secret

import (
	"path/filepath"
	"testing"
)

func newTestEncryptedStore(t *testing.T) *encryptedStore {
	t.Helper()
	dir := t.TempDir()
	return &encryptedStore{
		dataPath: filepath.Join(dir, "config.enc"),
		keyPath:  filepath.Join(dir, ".secret.key"),
	}
}

func TestEncryptedStoreRoundTrip(t *testing.T) {
	s := newTestEncryptedStore(t)

	if err := s.Set("work", "sk-super-secret"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get("work")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sk-super-secret" {
		t.Fatalf("Get = %q, want %q", got, "sk-super-secret")
	}

	if err := s.Delete("work"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get("work"); err == nil {
		t.Fatal("expected error reading deleted secret")
	}
}

func TestEncryptedStorePersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	a := &encryptedStore{dataPath: filepath.Join(dir, "config.enc"), keyPath: filepath.Join(dir, ".secret.key")}
	b := &encryptedStore{dataPath: filepath.Join(dir, "config.enc"), keyPath: filepath.Join(dir, ".secret.key")}

	if err := a.Set("personal", "tok-123"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := b.Get("personal")
	if err != nil {
		t.Fatalf("Get from second instance: %v", err)
	}
	if got != "tok-123" {
		t.Fatalf("Get = %q, want %q", got, "tok-123")
	}
}
