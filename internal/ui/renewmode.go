package ui

// renewMode governs what the dashboard does when a ChatGPT-OAuth profile's
// silent renewal (internal/codexrefresh) fails because its refresh_token
// itself is dead and only an interactive `codex login` can recover it.
// Cycled with the m key; see dashKeys.RenewMode.
type renewMode int

const (
	// renewAsk (default) surfaces a status hint naming the profile and
	// waits for the user to press l on it — the safest option, since it
	// never opens a browser without the user noticing.
	renewAsk renewMode = iota
	// renewAuto opens the ChatGPT OAuth browser flow immediately, with no
	// prompt, the moment a profile needs it.
	renewAuto
	// renewManual only updates the profile's badge; no status hint is
	// shown, so the user discovers it purely from the dashboard.
	renewManual
)

func (m renewMode) String() string {
	switch m {
	case renewAuto:
		return "auto"
	case renewManual:
		return "manual"
	default:
		return "ask"
	}
}

func (m renewMode) next() renewMode {
	return (m + 1) % 3
}
