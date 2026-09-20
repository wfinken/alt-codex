package secret

import (
	"errors"
	"testing"
)

type fakeStore struct {
	name    string
	data    map[string]string
	failSet bool
}

func newFakeStore(name string) *fakeStore {
	return &fakeStore{name: name, data: map[string]string{}}
}

func (f *fakeStore) Backend() string { return f.name }

func (f *fakeStore) Set(profileName, value string) error {
	if f.failSet {
		return errors.New("simulated backend rejection")
	}
	f.data[profileName] = value
	return nil
}

func (f *fakeStore) Get(profileName string) (string, error) {
	v, ok := f.data[profileName]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (f *fakeStore) Delete(profileName string) error {
	delete(f.data, profileName)
	return nil
}

func TestHybridStorePrefersPrimary(t *testing.T) {
	primary, fallback := newFakeStore("primary"), newFakeStore("fallback")
	h := &hybridStore{primary: primary, fallback: fallback}

	if err := h.Set("work", "secret-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, ok := fallback.data["work"]; ok {
		t.Error("fallback should be untouched when primary succeeds")
	}
	got, err := h.Get("work")
	if err != nil || got != "secret-value" {
		t.Fatalf("Get = %q, %v; want %q, nil", got, err, "secret-value")
	}
}

func TestHybridStoreFallsBackWhenPrimaryRejects(t *testing.T) {
	primary, fallback := newFakeStore("primary"), newFakeStore("fallback")
	primary.failSet = true
	h := &hybridStore{primary: primary, fallback: fallback}

	// Simulates a keychain that rejects an oversized OAuth credential blob.
	if err := h.Set("work", "huge-oauth-blob"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, ok := primary.data["work"]; ok {
		t.Error("primary should not hold the secret when its Set failed")
	}
	got, err := h.Get("work")
	if err != nil || got != "huge-oauth-blob" {
		t.Fatalf("Get = %q, %v; want %q, nil", got, err, "huge-oauth-blob")
	}
}

func TestHybridStoreDeleteRemovesFromBoth(t *testing.T) {
	primary, fallback := newFakeStore("primary"), newFakeStore("fallback")
	h := &hybridStore{primary: primary, fallback: fallback}

	primary.data["work"] = "a"
	fallback.data["work"] = "b" // simulate a leftover from an earlier fallback

	if err := h.Delete("work"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := h.Get("work"); err == nil {
		t.Error("expected Get to fail after Delete removed from both backends")
	}
}
