package ui

import (
	"fmt"
	"strings"
)

// renderHelp draws the "settings & shortcuts" overlay (? key). It exists so
// the dashboard itself can stay uncluttered: single-key toggles like
// auto-refresh (r) and renew mode (m) still act immediately from the
// dashboard, but their current values and the full key legend live here
// instead of in the always-visible header/footer.
func renderHelp(backend string, autoRefresh bool, renew renewMode) string {
	autoRefreshState := "off"
	if autoRefresh {
		autoRefreshState = "on"
	}

	lines := []string{
		formLabelStyle.Render("Settings & shortcuts"),
		"",
		keyHint("↑/k ↓/j", "navigate"),
		keyHint("enter/s", "switch to selected profile"),
		keyHint("a", "add a new account"),
		keyHint("d", "delete selected profile"),
		keyHint("r", fmt.Sprintf("toggle auto-refresh (currently %s)", autoRefreshState)),
		keyHint("l", "sign in / re-authenticate selected profile"),
		keyHint("m", fmt.Sprintf("cycle renew mode (currently %s)", renew)),
		keyHint("q", "quit"),
		"",
		formHintStyle.Render("secrets backend: " + backend),
		"",
		formHintStyle.Render("esc / ? / q   close"),
	}

	return helpBorderStyle.Render(strings.Join(lines, "\n"))
}
