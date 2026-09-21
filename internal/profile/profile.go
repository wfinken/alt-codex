// Package profile manages alt-codex profile metadata (FR-01, FR-03).
// Secret material never lives here — see internal/secret for that half
// of the split that keeps NFR-03 (no plaintext secrets on disk) true.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/wfinken/alt-codex/internal/paths"
)

// CredentialType describes how a profile's stored secret should be applied
// to the local Codex CLI configuration when the profile is activated.
type CredentialType string

const (
	// TypeAPIKey stores a bare API key/token string.
	TypeAPIKey CredentialType = "api_key"
	// TypeRawJSON stores a full, pre-formatted Codex auth.json payload.
	TypeRawJSON CredentialType = "raw_json"
)

// Profile is a single named Codex account's metadata. The secret itself is
// stored out-of-band (OS keychain or encrypted fallback) and referenced here
// only by the profile Name, which doubles as the secret store's lookup key.
type Profile struct {
	Name      string         `json:"name"`
	Type      CredentialType `json:"type"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	// LastSwitchedAt is zero if the profile has never been made active.
	LastSwitchedAt time.Time `json:"last_switched_at,omitempty"`
	// ExpiresAt is optional, user-supplied metadata (e.g. a known corporate
	// token rotation date) used to render the "Expired" dashboard badge.
	// alt-codex has no way to query Codex itself for token liveness.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Status summarizes a profile's dashboard badge.
type Status string

const (
	StatusActive Status = "Active"
	// StatusExpired covers non-OAuth (API key) profiles past their known
	// expiry: alt-codex can't renew these itself, only flag them.
	StatusExpired Status = "Expired"
	// StatusNeedsReauth covers ChatGPT-OAuth profiles whose token is past
	// its real expiry and whose refresh_token silent-renewal has failed
	// (or hasn't been attempted yet) — see internal/codexrefresh.
	StatusNeedsReauth Status = "NeedsReauth"
	StatusSaved       Status = "Saved"
)

// StatusOf reports p's dashboard badge given the store's active profile name.
func (p Profile) StatusOf(activeName string) Status {
	switch {
	case p.Name == activeName:
		return StatusActive
	case p.ExpiresAt != nil && p.ExpiresAt.Before(time.Now()):
		if p.Type == TypeRawJSON {
			return StatusNeedsReauth
		}
		return StatusExpired
	default:
		return StatusSaved
	}
}

type document struct {
	Active   string    `json:"active"`
	Profiles []Profile `json:"profiles"`
	// PromptIntegrationDisabled opts out of `alt-codex current` (roadmap:
	// "shell prompt integration"). Stored inverted, and omitted when false,
	// so profiles.json files written before this setting existed are
	// interpreted as enabled.
	PromptIntegrationDisabled bool `json:"prompt_integration_disabled,omitempty"`
}

// Store persists profile metadata to profiles.json (FR-03).
type Store struct {
	path string
}

// NewStore opens (without yet reading) the profile store at its default path.
func NewStore() (*Store, error) {
	p, err := paths.ProfilesFile()
	if err != nil {
		return nil, err
	}
	return &Store{path: p}, nil
}

func (s *Store) load() (document, error) {
	doc := document{}
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return doc, nil
	}
	if err != nil {
		return doc, err
	}
	if len(b) == 0 {
		return doc, nil
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return doc, fmt.Errorf("parse %s: %w", s.path, err)
	}
	return doc, nil
}

func (s *Store) save(doc document) error {
	sort.Slice(doc.Profiles, func(i, j int) bool {
		return doc.Profiles[i].Name < doc.Profiles[j].Name
	})
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// List returns all profiles and the name of the active profile (which may
// be empty if none has been activated yet).
func (s *Store) List() ([]Profile, string, error) {
	doc, err := s.load()
	if err != nil {
		return nil, "", err
	}
	return doc.Profiles, doc.Active, nil
}

// Get returns the profile with the given name.
func (s *Store) Get(name string) (Profile, error) {
	doc, err := s.load()
	if err != nil {
		return Profile{}, err
	}
	for _, p := range doc.Profiles {
		if p.Name == name {
			return p, nil
		}
	}
	return Profile{}, fmt.Errorf("no profile named %q", name)
}

// Add inserts a new profile. It returns an error if the name is already taken.
func (s *Store) Add(name string, typ CredentialType, expiresAt *time.Time) (Profile, error) {
	doc, err := s.load()
	if err != nil {
		return Profile{}, err
	}
	for _, p := range doc.Profiles {
		if p.Name == name {
			return Profile{}, fmt.Errorf("profile %q already exists", name)
		}
	}
	now := time.Now().UTC()
	p := Profile{Name: name, Type: typ, CreatedAt: now, UpdatedAt: now, ExpiresAt: expiresAt}
	doc.Profiles = append(doc.Profiles, p)
	if err := s.save(doc); err != nil {
		return Profile{}, err
	}
	return p, nil
}

// SetActive marks name as the active profile (FR-04, FR-05).
func (s *Store) SetActive(name string) error {
	doc, err := s.load()
	if err != nil {
		return err
	}
	found := false
	now := time.Now().UTC()
	for i, p := range doc.Profiles {
		if p.Name == name {
			doc.Profiles[i].LastSwitchedAt = now
			found = true
		}
	}
	if !found {
		return fmt.Errorf("no profile named %q", name)
	}
	doc.Active = name
	return s.save(doc)
}

// SetExpiresAt updates name's cached expiry metadata. It exists mainly for
// the codexrefresh flow: once a ChatGPT-OAuth profile is refreshed (silently
// or interactively), the token's own exp claim is the authoritative expiry,
// replacing whatever was known before.
func (s *Store) SetExpiresAt(name string, exp *time.Time) error {
	doc, err := s.load()
	if err != nil {
		return err
	}
	found := false
	now := time.Now().UTC()
	for i, p := range doc.Profiles {
		if p.Name == name {
			doc.Profiles[i].ExpiresAt = exp
			doc.Profiles[i].UpdatedAt = now
			found = true
		}
	}
	if !found {
		return fmt.Errorf("no profile named %q", name)
	}
	return s.save(doc)
}

// PromptIntegrationEnabled reports whether `alt-codex current` should print
// the active profile's name for shell prompt hooks (roadmap: "shell prompt
// integration"). Enabled by default; toggled from the dashboard's p key.
func (s *Store) PromptIntegrationEnabled() (bool, error) {
	doc, err := s.load()
	if err != nil {
		return false, err
	}
	return !doc.PromptIntegrationDisabled, nil
}

// SetPromptIntegrationEnabled persists the p-key toggle. It has to live on
// disk, not just in the running TUI's model, since `alt-codex current` reads
// it from a separate process invocation the TUI has no other way to reach.
func (s *Store) SetPromptIntegrationEnabled(enabled bool) error {
	doc, err := s.load()
	if err != nil {
		return err
	}
	doc.PromptIntegrationDisabled = !enabled
	return s.save(doc)
}

// ImportResult reports what Import did with each profile it was given.
type ImportResult struct {
	Imported []string
	// Skipped lists profiles that already existed and were left untouched
	// because overwrite was false.
	Skipped []string
}

// Import adds profiles from a decrypted backup.Archive, preserving their
// original timestamps rather than stamping new ones the way Add does — an
// import is a restore, not a fresh account. A profile whose name already
// exists is left alone unless overwrite is true, in which case its metadata
// is replaced outright. The store's active pointer is never touched: which
// profile Codex is currently pointed at shouldn't change just from restoring
// a backup.
func (s *Store) Import(profiles []Profile, overwrite bool) (ImportResult, error) {
	doc, err := s.load()
	if err != nil {
		return ImportResult{}, err
	}

	index := make(map[string]int, len(doc.Profiles))
	for i, p := range doc.Profiles {
		index[p.Name] = i
	}

	var res ImportResult
	for _, p := range profiles {
		if i, exists := index[p.Name]; exists {
			if !overwrite {
				res.Skipped = append(res.Skipped, p.Name)
				continue
			}
			doc.Profiles[i] = p
		} else {
			index[p.Name] = len(doc.Profiles)
			doc.Profiles = append(doc.Profiles, p)
		}
		res.Imported = append(res.Imported, p.Name)
	}

	if len(res.Imported) == 0 {
		return res, nil
	}
	return res, s.save(doc)
}

// Remove deletes the named profile's metadata. If it was the active profile,
// the store's active pointer is cleared. Removing the associated secret from
// the secret store is the caller's responsibility (FR-08).
func (s *Store) Remove(name string) error {
	doc, err := s.load()
	if err != nil {
		return err
	}
	out := doc.Profiles[:0]
	found := false
	for _, p := range doc.Profiles {
		if p.Name == name {
			found = true
			continue
		}
		out = append(out, p)
	}
	if !found {
		return fmt.Errorf("no profile named %q", name)
	}
	doc.Profiles = out
	if doc.Active == name {
		doc.Active = ""
	}
	return s.save(doc)
}
