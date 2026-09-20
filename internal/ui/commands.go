package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wfinken/alt-codex/internal/codexconfig"
	"github.com/wfinken/alt-codex/internal/codexrefresh"
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
		// A ChatGPT-OAuth credential's own access token carries its real
		// expiry; prefer that over whatever (if anything) the user typed,
		// so the dashboard's badge is accurate from the moment it's added.
		if typ == profile.TypeRawJSON {
			if exp, ok := codexrefresh.ExpiryOf(token); ok {
				expiresAt = &exp
			}
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

// autoRefreshInterval is how often the dashboard reloads profiles and
// re-runs the token-renewal sweep while auto-refresh is enabled (roadmap:
// "auto-refresh tokens nearing expiration"). Reloading periodically
// re-evaluates each profile's ExpiresAt against the current time, so a
// profile's badge updates on its own instead of only on the next manual
// action.
const autoRefreshInterval = 30 * time.Second

type autoRefreshTickMsg struct{}

func autoRefreshTick() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(time.Time) tea.Msg { return autoRefreshTickMsg{} })
}

// renewResult is one profile's outcome from a renewCheckCmd sweep.
type renewResult struct {
	profile   string
	refreshed bool // silent renewal succeeded
	needsAuth bool // refresh_token is dead; only interactive login can fix it
	err       error
}

type renewCheckMsg struct {
	results []renewResult
}

// renewCheckCmd sweeps every ChatGPT-OAuth profile whose cached expiry is
// unknown or within codexrefresh.NearExpiryWindow and attempts a silent
// renewal (internal/codexrefresh). It runs on app startup and, while
// auto-refresh is on, on every autoRefreshTick — see the roadmap item
// "auto-refresh tokens nearing expiration".
func renewCheckCmd(store *profile.Store, secrets secret.Store) tea.Cmd {
	return func() tea.Msg {
		items, _, err := store.List()
		if err != nil {
			return renewCheckMsg{}
		}

		now := time.Now()
		var results []renewResult
		for _, p := range items {
			if p.Type != profile.TypeRawJSON {
				continue
			}
			if p.ExpiresAt != nil && p.ExpiresAt.After(now.Add(codexrefresh.NearExpiryWindow)) {
				continue
			}

			authJSON, err := secrets.Get(p.Name)
			if err != nil {
				results = append(results, renewResult{profile: p.Name, err: err})
				continue
			}

			if p.ExpiresAt == nil {
				// Backfill from the token's own claim first — cheap, no
				// network call, no refresh_token rotation — and only fall
				// through to a live renewal below if that still shows it's
				// actually near/at expiry.
				if exp, ok := codexrefresh.ExpiryOf(authJSON); ok {
					_ = store.SetExpiresAt(p.Name, &exp)
					if exp.After(now.Add(codexrefresh.NearExpiryWindow)) {
						continue
					}
				}
			}

			newJSON, refreshed, err := codexrefresh.Try(authJSON)
			if err != nil {
				results = append(results, renewResult{profile: p.Name, err: err})
				continue
			}
			// Persist unconditionally: the refresh_token rotates on every
			// attempt, so even a failed renewal leaves the previously
			// stored secret's refresh_token dead.
			_ = secrets.Set(p.Name, newJSON)
			if exp, ok := codexrefresh.ExpiryOf(newJSON); ok {
				_ = store.SetExpiresAt(p.Name, &exp)
			}
			results = append(results, renewResult{profile: p.Name, refreshed: refreshed, needsAuth: !refreshed})
		}
		return renewCheckMsg{results: results}
	}
}

// reauthSavedMsg reports the outcome of persisting a profile's freshly
// captured credential after an interactive re-login (reauthDialog).
type reauthSavedMsg struct {
	name string
	err  error
}

func saveReauthCmd(store *profile.Store, secrets secret.Store, name, authJSON string) tea.Cmd {
	return func() tea.Msg {
		if err := secrets.Set(name, authJSON); err != nil {
			return reauthSavedMsg{name: name, err: err}
		}
		if exp, ok := codexrefresh.ExpiryOf(authJSON); ok {
			_ = store.SetExpiresAt(name, &exp)
		}
		return reauthSavedMsg{name: name}
	}
}
