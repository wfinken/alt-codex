package profile

import (
	"path/filepath"
	"testing"
	"time"
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

func TestStatusOfExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)

	apiKey := Profile{Name: "work", Type: TypeAPIKey, ExpiresAt: &past}
	if got := apiKey.StatusOf(""); got != StatusExpired {
		t.Errorf("api key StatusOf = %q, want %q", got, StatusExpired)
	}

	oauth := Profile{Name: "personal", Type: TypeRawJSON, ExpiresAt: &past}
	if got := oauth.StatusOf(""); got != StatusNeedsReauth {
		t.Errorf("oauth StatusOf = %q, want %q", got, StatusNeedsReauth)
	}
}

func TestSetExpiresAt(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Add("work", TypeRawJSON, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}

	exp := time.Now().Add(24 * time.Hour)
	if err := s.SetExpiresAt("work", &exp); err != nil {
		t.Fatalf("SetExpiresAt: %v", err)
	}
	p, err := s.Get("work")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.ExpiresAt == nil || !p.ExpiresAt.Equal(exp) {
		t.Fatalf("ExpiresAt = %v, want %v", p.ExpiresAt, exp)
	}

	if err := s.SetExpiresAt("ghost", &exp); err == nil {
		t.Fatal("expected error setting expiry on unknown profile")
	}
}

func TestImport(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Add("work", TypeAPIKey, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.SetActive("work"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	old := time.Now().Add(-30 * 24 * time.Hour).UTC()
	incoming := []Profile{
		{Name: "work", Type: TypeAPIKey, CreatedAt: old, UpdatedAt: old},      // conflicts
		{Name: "personal", Type: TypeRawJSON, CreatedAt: old, UpdatedAt: old}, // new
	}

	res, err := s.Import(incoming, false)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(res.Imported) != 1 || res.Imported[0] != "personal" {
		t.Fatalf("Imported = %v, want [personal]", res.Imported)
	}
	if len(res.Skipped) != 1 || res.Skipped[0] != "work" {
		t.Fatalf("Skipped = %v, want [work]", res.Skipped)
	}

	p, err := s.Get("work")
	if err != nil {
		t.Fatalf("Get(work): %v", err)
	}
	if p.CreatedAt.Equal(old) {
		t.Fatal("existing profile work was overwritten despite overwrite=false")
	}

	_, active, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if active != "work" {
		t.Fatalf("active = %q after import, want unchanged %q", active, "work")
	}

	res, err = s.Import(incoming, true)
	if err != nil {
		t.Fatalf("Import with overwrite: %v", err)
	}
	if len(res.Imported) != 2 {
		t.Fatalf("Imported = %v, want both profiles overwritten/added", res.Imported)
	}
	if len(res.Skipped) != 0 {
		t.Fatalf("Skipped = %v, want none with overwrite=true", res.Skipped)
	}

	p, err = s.Get("work")
	if err != nil {
		t.Fatalf("Get(work): %v", err)
	}
	if !p.CreatedAt.Equal(old) {
		t.Fatalf("work.CreatedAt = %v after overwrite, want preserved import timestamp %v", p.CreatedAt, old)
	}
}

func TestPromptIntegrationEnabled(t *testing.T) {
	s := newTestStore(t)

	enabled, err := s.PromptIntegrationEnabled()
	if err != nil {
		t.Fatalf("PromptIntegrationEnabled: %v", err)
	}
	if !enabled {
		t.Fatal("PromptIntegrationEnabled on a fresh store = false, want true (enabled by default)")
	}

	if err := s.SetPromptIntegrationEnabled(false); err != nil {
		t.Fatalf("SetPromptIntegrationEnabled(false): %v", err)
	}
	if enabled, err = s.PromptIntegrationEnabled(); err != nil {
		t.Fatalf("PromptIntegrationEnabled: %v", err)
	} else if enabled {
		t.Fatal("PromptIntegrationEnabled = true after disabling, want false")
	}

	if err := s.SetPromptIntegrationEnabled(true); err != nil {
		t.Fatalf("SetPromptIntegrationEnabled(true): %v", err)
	}
	if enabled, err = s.PromptIntegrationEnabled(); err != nil {
		t.Fatalf("PromptIntegrationEnabled: %v", err)
	} else if !enabled {
		t.Fatal("PromptIntegrationEnabled = false after re-enabling, want true")
	}
}
