package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wfinken/alt-codex/internal/codexlogin"
)

// reauthDialog drives an interactive ChatGPT-OAuth re-login for an existing
// profile whose refresh_token has died (a silent renewal via codexrefresh
// was tried first and failed). It reuses the real Codex CLI's own
// `codex login` browser flow — the same mechanism the add form's ctrl+l
// path uses — but updates an existing profile's secret in place instead of
// creating a new one.
type reauthDialog struct {
	profile string
	spinner spinner.Model
	sess    *codexlogin.Session
	lines   []string
	done    bool
	err     string
}

func newReauthDialog(name string) reauthDialog {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = statusOKStyle
	return reauthDialog{profile: name, spinner: sp, sess: codexlogin.Start()}
}

// cancel aborts the in-flight login attempt, if any. Called when the user
// backs out with esc.
func (d *reauthDialog) cancel() {
	if d.sess != nil {
		d.sess.Cancel()
	}
}

// reauthDoneMsg is emitted (via Update's third return) once the login
// session finishes successfully; the caller is responsible for persisting
// authJSON as profile's new secret.
type reauthDoneMsg struct {
	profile  string
	authJSON string
}

func (d reauthDialog) Update(msg tea.Msg) (reauthDialog, tea.Cmd, *reauthDoneMsg) {
	switch m := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		d.spinner, cmd = d.spinner.Update(m)
		return d, cmd, nil
	case loginTickMsg:
		lines, done, authJSON, err := d.sess.Snapshot()
		d.lines = lines
		if !done {
			return d, pollLoginCmd(), nil
		}
		d.done = true
		if err != nil {
			d.err = err.Error()
			return d, nil, nil
		}
		return d, nil, &reauthDoneMsg{profile: d.profile, authJSON: authJSON}
	}
	return d, nil, nil
}

func (d reauthDialog) View() string {
	lines := []string{
		titleStyle.Render(" Re-authenticate " + d.profile + " "),
		"",
	}
	switch {
	case d.err != "":
		lines = append(lines, statusErrStyle.Render("✗ codex login: "+d.err))
		lines = append(lines, formHintStyle.Render("esc to go back"))
	case d.done:
		lines = append(lines, statusOKStyle.Render("✓ signed in — saving…"))
	default:
		lines = append(lines, fmt.Sprintf("%s waiting for you to finish in the browser…", d.spinner.View()))
		for _, l := range lastN(d.lines, 6) {
			lines = append(lines, formHintStyle.Render("  "+l))
		}
		lines = append(lines, formHintStyle.Render("esc to cancel"))
	}
	return appPadding.Render(strings.Join(lines, "\n"))
}
