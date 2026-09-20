package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// confirmDialog implements the destructive-action prompts required by
// FR-08/FR-09 (delete a profile; extra warning when it's the active one).
type confirmDialog struct {
	target   string
	message  string
	isActive bool
	yes      bool // which choice is highlighted; default "no" to avoid slips
}

func newConfirmDialog(target, message string, isActive bool) confirmDialog {
	return confirmDialog{target: target, message: message, isActive: isActive}
}

type confirmResultMsg struct {
	target    string
	confirmed bool
}

func (c confirmDialog) Update(msg tea.Msg) (confirmDialog, *confirmResultMsg) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	switch km.String() {
	case "left", "h", "right", "l", "tab":
		c.yes = !c.yes
	case "y":
		return c, &confirmResultMsg{target: c.target, confirmed: true}
	case "n", "esc":
		return c, &confirmResultMsg{target: c.target, confirmed: false}
	case "enter":
		return c, &confirmResultMsg{target: c.target, confirmed: c.yes}
	}
	return c, nil
}

func (c confirmDialog) View() string {
	var yes, no string
	if c.yes {
		yes = selectedRowStyle.Render("▸ yes")
		no = "  no"
	} else {
		yes = "  yes"
		no = selectedRowStyle.Render("▸ no")
	}

	lines := []string{c.message}
	if c.isActive {
		lines = append(lines, "", statusErrStyle.Render("⚠ this is your ACTIVE profile — Codex will lose access to it."))
	}
	lines = append(lines, "", yes+"    "+no, "", formHintStyle.Render("←/→ choose · enter confirm · y/n · esc cancel"))

	return dialogBorderStyle.Render(strings.Join(lines, "\n"))
}
