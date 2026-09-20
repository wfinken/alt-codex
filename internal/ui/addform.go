package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wfinken/alt-codex/internal/codexlogin"
	"github.com/wfinken/alt-codex/internal/profile"
)

type addField int

const (
	fieldName addField = iota
	fieldToken
	fieldExpires
	fieldCount
)

// addForm is the "Add Account" view: name + token/credential fields, an
// optional known-expiry date, and a spinner shown while the submission
// command (validate + persist) is in flight (PRD §4, Add Account View).
// It also supports capturing a ChatGPT-OAuth (non-API-key) credential
// directly from the real Codex CLI's own login flow — see codexlogin.
type addForm struct {
	inputs   [fieldCount]textinput.Model
	credType profile.CredentialType
	focus    addField
	spinner  spinner.Model

	submitting bool
	errMsg     string

	loggingIn      bool
	loginSucceeded bool
	loginErr       string
	loginLines     []string
	loginSess      *codexlogin.Session
}

func newAddForm() addForm {
	name := textinput.New()
	name.Placeholder = "work-corp"
	name.Focus()
	name.CharLimit = 64

	token := textinput.New()
	token.Placeholder = "paste API key or Codex auth.json"
	token.EchoMode = textinput.EchoPassword
	token.EchoCharacter = '•'
	token.CharLimit = 8192

	expires := textinput.New()
	expires.Placeholder = "YYYY-MM-DD (optional)"
	expires.CharLimit = 10

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = statusOKStyle

	return addForm{
		inputs:   [fieldCount]textinput.Model{fieldName: name, fieldToken: token, fieldExpires: expires},
		credType: profile.TypeAPIKey,
		spinner:  sp,
	}
}

// submitAddMsg is emitted (as a tea.Cmd's return value bubbled up via
// Update's second return) when the user confirms the form.
type submitAddMsg struct {
	name, token, expires string
	credType             profile.CredentialType
}

// loginTickMsg drives polling of an in-flight codexlogin.Session.
type loginTickMsg struct{}

func pollLoginCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return loginTickMsg{} })
}

func (f addForm) updateFocus() addForm {
	for i := range f.inputs {
		if addField(i) == f.focus {
			f.inputs[i].Focus()
			f.inputs[i].PromptStyle = formLabelStyle
		} else {
			f.inputs[i].Blur()
		}
	}
	return f
}

// cancelLogin aborts an in-flight codex login attempt, if any. Exported for
// the root model to call when the user backs out of the add form with esc.
func (f *addForm) cancelLogin() {
	if f.loginSess != nil {
		f.loginSess.Cancel()
	}
}

func (f addForm) Update(msg tea.Msg) (addForm, tea.Cmd, *submitAddMsg) {
	if f.submitting || f.loggingIn {
		switch m := msg.(type) {
		case spinner.TickMsg:
			var cmd tea.Cmd
			f.spinner, cmd = f.spinner.Update(m)
			return f, cmd, nil
		case loginTickMsg:
			lines, done, err, authJSON := f.loginSess.Snapshot()
			f.loginLines = lines
			if !done {
				return f, pollLoginCmd(), nil
			}
			f.loggingIn = false
			if err != nil {
				f.loginErr = err.Error()
				return f, nil, nil
			}
			f.loginErr = ""
			f.loginSucceeded = true
			f.credType = profile.TypeRawJSON
			f.inputs[fieldToken].SetValue(authJSON)
			f.inputs[fieldToken].EchoMode = textinput.EchoNormal
			f.inputs[fieldToken].Placeholder = `{"OPENAI_API_KEY": "..."}`
			return f, nil, nil
		}
		return f, nil, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			f.focus = (f.focus + 1) % fieldCount
			f = f.updateFocus()
			return f, nil, nil
		case "shift+tab", "up":
			f.focus = (f.focus - 1 + fieldCount) % fieldCount
			f = f.updateFocus()
			return f, nil, nil
		case "ctrl+t":
			if f.credType == profile.TypeAPIKey {
				f.credType = profile.TypeRawJSON
				f.inputs[fieldToken].EchoMode = textinput.EchoNormal
				f.inputs[fieldToken].Placeholder = `{"OPENAI_API_KEY": "..."}`
			} else {
				f.credType = profile.TypeAPIKey
				f.inputs[fieldToken].EchoMode = textinput.EchoPassword
				f.inputs[fieldToken].Placeholder = "paste API key or Codex auth.json"
			}
			return f, nil, nil
		case "ctrl+l":
			f.loggingIn = true
			f.loginSucceeded = false
			f.errMsg = ""
			f.loginErr = ""
			f.loginLines = nil
			f.loginSess = codexlogin.Start()
			return f, tea.Batch(f.spinner.Tick, pollLoginCmd()), nil
		case "enter":
			if f.focus != fieldExpires {
				f.focus++
				f = f.updateFocus()
				return f, nil, nil
			}
			f.submitting = true
			f.errMsg = ""
			sub := submitAddMsg{
				name:     f.inputs[fieldName].Value(),
				token:    f.inputs[fieldToken].Value(),
				expires:  f.inputs[fieldExpires].Value(),
				credType: f.credType,
			}
			return f, f.spinner.Tick, &sub
		}
	}

	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return f, cmd, nil
}

func (f addForm) View() string {
	typeLabel := "API key"
	if f.credType == profile.TypeRawJSON {
		typeLabel = "raw auth.json"
	}

	lines := []string{
		titleStyle.Render(" Add Codex Account "),
		"",
		formHintStyle.Render("tab/shift+tab move · ctrl+t toggle type · ctrl+l sign in with Codex CLI · enter next/submit · esc cancel"),
		"",
		formLabelStyle.Render("Credential type: ") + typeLabel,
		"",
		formLabelStyle.Render("Profile name"),
		f.inputs[fieldName].View(),
		"",
		formLabelStyle.Render("Token / credential"),
		f.inputs[fieldToken].View(),
		"",
		formLabelStyle.Render("Known expiry (optional)"),
		f.inputs[fieldExpires].View(),
		"",
	}

	switch {
	case f.loggingIn:
		lines = append(lines, fmt.Sprintf("%s signing in via codex login — waiting for you to finish in the browser…", f.spinner.View()))
		for _, l := range lastN(f.loginLines, 6) {
			lines = append(lines, formHintStyle.Render("  "+l))
		}
		lines = append(lines, formHintStyle.Render("esc to cancel"))
	case f.loginErr != "":
		lines = append(lines, statusErrStyle.Render("✗ codex login: "+f.loginErr))
	case f.loginSucceeded:
		lines = append(lines, statusOKStyle.Render("✓ signed in via Codex CLI — credential captured below, press enter to save"))
	}

	if f.submitting {
		lines = append(lines, fmt.Sprintf("%s validating & saving…", f.spinner.View()))
	}
	if f.errMsg != "" {
		lines = append(lines, statusErrStyle.Render("✗ "+f.errMsg))
	}

	return appPadding.Render(strings.Join(lines, "\n"))
}

func lastN(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
