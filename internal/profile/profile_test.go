package profile

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{path: filepath.Join(t.TempDir(), "profiles.json")}
}

func TestAddListGetRemove(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Add("work", TypeAPIKey, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := s.Add("work", TypeAPIKey, nil); err == nil {
		t.Fatal("expected error adding duplicate profile name")
	}

	items, active, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || active != "" {
		t.Fatalf("got items=%v active=%q, want 1 item and no active profile", items, active)
	}

	if _, err := s.Get("work"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, err := s.Get("missing"); err == nil {
		t.Fatal("expected error getting unknown profile")
	}

	if err := s.Remove("work"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := s.Remove("work"); err == nil {
		t.Fatal("expected error removing already-removed profile")
	}
}

func TestSetActiveClearsOnRemove(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Add("work", TypeAPIKey, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.SetActive("work"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	_, active, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if active != "work" {
		t.Fatalf("active = %q, want %q", active, "work")
	}

	if err := s.Remove("work"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, active, err = s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if active != "" {
		t.Fatalf("active = %q after removing active profile, want empty", active)
	}
}

func TestSetActiveUnknownProfile(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetActive("ghost"); err == nil {
		t.Fatal("expected error activating unknown profile")
	}
}

func TestStatusOf(t *testing.T) {
	p := Profile{Name: "work"}
	if got := p.StatusOf("work"); got != StatusActive {
		t.Errorf("StatusOf active = %q, want %q", got, StatusActive)
	}
	if got := p.StatusOf("other"); got != StatusSaved {
		t.Errorf("StatusOf saved = %q, want %q", got, StatusSaved)
	}
}
