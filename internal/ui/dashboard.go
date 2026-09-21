package ui

import (
	"fmt"
	"strings"

	"github.com/wfinken/alt-codex/internal/profile"
)

func badgeFor(status profile.Status) string {
	switch status {
	case profile.StatusActive:
		return activeBadge.Render("● Active")
	case profile.StatusExpired:
		return expiredBadge.Render("✕ Expired")
	case profile.StatusNeedsReauth:
		return expiredBadge.Render("⟳ Needs re-auth")
	default:
		return savedBadge.Render("○ Saved")
	}
}

// dashboardSettings bundles the dashboard's toggleable state — shown as the
// de-emphasized options line and, in full, in the ? overlay (help.go) — so
// renderDashboard and renderHelp don't grow another positional bool for
// every setting added to the roadmap.
type dashboardSettings struct {
	backend           string
	autoRefresh       bool
	renewMode         renewMode
	promptIntegration bool
}

func renderDashboard(items []profile.Profile, active string, cursor int, s dashboardSettings) string {
	var b strings.Builder

	header := "⌁ alt-codex"
	activeLine := subtitleStyle.Render("no active profile")
	if active != "" {
		activeLine = subtitleStyle.Render("active: ") + activeBadge.Render(active)
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("  ")
	b.WriteString(activeLine)
	b.WriteString("\n\n")

	if len(items) == 0 {
		b.WriteString(formHintStyle.Render("No profiles yet. Press "))
		b.WriteString(footerKeyStyle.Render("a"))
		b.WriteString(formHintStyle.Render(" to add your first Codex account."))
		b.WriteString("\n")
	}

	for i, p := range items {
		cur := "  "
		style := rowStyle
		if i == cursor {
			cur = cursorStyle.Render("▸ ")
			style = selectedRowStyle
		}
		name := style.Render(p.Name)
		row := fmt.Sprintf("%s%-28s %s", cur, name, badgeFor(p.StatusOf(active)))
		if !p.LastSwitchedAt.IsZero() {
			row += "  " + formHintStyle.Render("last switched "+p.LastSwitchedAt.Local().Format("2006-01-02 15:04"))
		}
		b.WriteString(row)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(formHintStyle.Render(optionsSummary(s)))
	b.WriteString("\n")
	footer := strings.Join([]string{
		keyHint("↑/k ↓/j", "navigate"),
		keyHint("enter/s", "switch"),
		keyHint("a", "add"),
		keyHint("d", "delete"),
		keyHint("?", "settings & shortcuts"),
		keyHint("q", "quit"),
	}, "   ")
	b.WriteString(footerStyle.Render(footer))

	return appPadding.Render(b.String())
}

// optionsSummary renders the dashboard's de-emphasized settings line: just
// enough to see current state at a glance, with the full explanation and key
// legend pushed into the ? overlay (see help.go) to keep the main view
// uncluttered.
func optionsSummary(s dashboardSettings) string {
	return fmt.Sprintf("options: %s · auto-refresh %s · renew mode %s · shell prompt %s",
		s.backend, onOff(s.autoRefresh), s.renewMode, onOff(s.promptIntegration))
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
