package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary   = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"}
	colorAccent    = lipgloss.AdaptiveColor{Light: "#0EA5E9", Dark: "#38BDF8"}
	colorActive    = lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"}
	colorExpired   = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}
	colorSaved     = lipgloss.AdaptiveColor{Light: "#71717A", Dark: "#A1A1AA"}
	colorMuted     = lipgloss.AdaptiveColor{Light: "#A1A1AA", Dark: "#71717A"}
	colorErrorText = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(colorPrimary).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	activeBadge  = lipgloss.NewStyle().Bold(true).Foreground(colorActive)
	savedBadge   = lipgloss.NewStyle().Foreground(colorSaved)
	expiredBadge = lipgloss.NewStyle().Bold(true).Foreground(colorExpired)

	cursorStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	rowStyle         = lipgloss.NewStyle()

	footerStyle    = lipgloss.NewStyle().Foreground(colorMuted).MarginTop(1)
	footerKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	statusOKStyle  = lipgloss.NewStyle().Foreground(colorActive)
	statusErrStyle = lipgloss.NewStyle().Foreground(colorErrorText)

	formLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	formHintStyle  = lipgloss.NewStyle().Foreground(colorMuted)

	dialogBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorExpired).
				Padding(1, 3)

	appPadding = lipgloss.NewStyle().Padding(1, 2)
)

func keyHint(key, label string) string {
	return footerKeyStyle.Render(key) + " " + label
}
