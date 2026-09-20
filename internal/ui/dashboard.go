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

func renderDashboard(items []profile.Profile, active string, cursor int, backend string, autoRefresh bool, renew renewMode, width int) string {
	var b strings.Builder

	header := "⌁ alt-codex"
	activeLine := subtitleStyle.Render("no active profile")
	if active != "" {
		activeLine = subtitleStyle.Render("active: ") + activeBadge.Render(active)
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("  ")
	b.WriteString(activeLine)
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(fmt.Sprintf("secrets backed by: %s", backend)))
	b.WriteString("  ")
	b.WriteString(autoRefreshIndicator(autoRefresh))
	b.WriteString("  ")
	b.WriteString(subtitleStyle.Render(fmt.Sprintf("renew mode: %s", renew)))
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
	footer := strings.Join([]string{
		keyHint("↑/k ↓/j", "navigate"),
		keyHint("enter/s", "switch"),
		keyHint("a", "add"),
		keyHint("d", "delete"),
		keyHint("r", "auto-refresh"),
		keyHint("l", "sign in"),
		keyHint("m", "renew mode"),
		keyHint("q", "quit"),
	}, "   ")
	b.WriteString(footerStyle.Render(footer))

	return appPadding.Render(b.String())
}

// autoRefreshIndicator renders the dashboard's current auto-refresh state
// (roadmap: "auto-refresh tokens nearing expiration"), toggled via the r key.
func autoRefreshIndicator(on bool) string {
	if on {
		return activeBadge.Render("↻ auto-refresh on")
	}
	return formHintStyle.Render("↻ auto-refresh off")
}
