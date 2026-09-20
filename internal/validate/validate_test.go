package validate

import (
	"testing"

	"github.com/wfinken/alt-codex/internal/profile"
)

func TestProfileName(t *testing.T) {
	cases := map[string]bool{
		"work-corp": true,
		"":          false,
		" work":     false,
		"a/b":       false,
	}
	for name, want := range cases {
		if err := ProfileName(name); (err == nil) != want {
			t.Errorf("ProfileName(%q) err=%v, want valid=%v", name, err, want)
		}
	}
}

func TestCredentialAPIKey(t *testing.T) {
	if err := Credential(profile.TypeAPIKey, ""); err == nil {
		t.Error("expected error for empty key")
	}
	if err := Credential(profile.TypeAPIKey, "short"); err == nil {
		t.Error("expected error for too-short key")
	}
	if err := Credential(profile.TypeAPIKey, "sk-1234567890abcdef"); err != nil {
		t.Errorf("expected valid key to pass, got %v", err)
	}
}

func TestCredentialRawJSON(t *testing.T) {
	if err := Credential(profile.TypeRawJSON, "not json"); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if err := Credential(profile.TypeRawJSON, `{"OPENAI_API_KEY":"x"}`); err != nil {
		t.Errorf("expected valid JSON to pass, got %v", err)
	}
}

func TestExpiresAt(t *testing.T) {
	if t2, err := ExpiresAt(""); err != nil || t2 != nil {
		t.Errorf("ExpiresAt(\"\") = %v, %v; want nil, nil", t2, err)
	}
	if _, err := ExpiresAt("not-a-date"); err == nil {
		t.Error("expected error for malformed date")
	}
	if t2, err := ExpiresAt("2026-01-02"); err != nil || t2 == nil {
		t.Errorf("ExpiresAt valid date failed: %v, %v", t2, err)
	}
}
