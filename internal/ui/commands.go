package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wfinken/alt-codex/internal/codexconfig"
	"github.com/wfinken/alt-codex/internal/profile"
	"github.com/wfinken/alt-codex/internal/secret"
	"github.com/wfinken/alt-codex/internal/validate"
)

type profilesLoadedMsg struct {
	items  []profile.Profile
	active string
	err    error
}

type switchedMsg struct {
	name string
	err  error
}

type addedMsg struct {
	p   profile.Profile
	err error
}

type deletedMsg struct {
	name      string
	wasActive bool
	err       error
}

func loadProfilesCmd(store *profile.Store) tea.Cmd {
	return func() tea.Msg {
		items, active, err := store.List()
		return profilesLoadedMsg{items: items, active: active, err: err}
	}
}

// switchCmd implements FR-04: pull the profile's secret, write it to the
// local Codex CLI auth file, then record it as active in profiles.json.
func switchCmd(store *profile.Store, secrets secret.Store, name string) tea.Cmd {
	return func() tea.Msg {
		p, err := store.Get(name)
		if err != nil {
			return switchedMsg{name: name, err: err}
		}
		val, err := secrets.Get(name)
		if err != nil {
			return switchedMsg{name: name, err: err}
		}
		if err := codexconfig.Apply(p.Type, val); err != nil {
			return switchedMsg{name: name, err: err}
		}
		if err := store.SetActive(name); err != nil {
			return switchedMsg{name: name, err: err}
		}
		return switchedMsg{name: name}
	}
}

// addCmd implements FR-06/FR-07: validate, then persist metadata and secret
// together, rolling the metadata back if the secret write fails so the two
// stores never disagree about which profiles exist.
func addCmd(store *profile.Store, secrets secret.Store, name string, typ profile.CredentialType, token string, expiresRaw string) tea.Cmd {
	return func() tea.Msg {
		if err := validate.ProfileName(name); err != nil {
			return addedMsg{err: err}
		}
		if err := validate.Credential(typ, token); err != nil {
			return addedMsg{err: err}
		}
		expiresAt, err := validate.ExpiresAt(expiresRaw)
		if err != nil {
			return addedMsg{err: err}
		}

		p, err := store.Add(name, typ, expiresAt)
		if err != nil {
			return addedMsg{err: err}
		}
		if err := secrets.Set(name, token); err != nil {
			_ = store.Remove(name)
			return addedMsg{err: err}
		}
		return addedMsg{p: p}
	}
}

// deleteCmd implements FR-08: purge the secret first, then the metadata.
func deleteCmd(store *profile.Store, secrets secret.Store, name string, wasActive bool) tea.Cmd {
	return func() tea.Msg {
		if err := secrets.Delete(name); err != nil {
			return deletedMsg{name: name, err: err}
		}
		if err := store.Remove(name); err != nil {
			return deletedMsg{name: name, err: err}
		}
		return deletedMsg{name: name, wasActive: wasActive}
	}
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

type clearStatusMsg struct{}
