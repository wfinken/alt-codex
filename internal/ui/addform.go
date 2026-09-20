package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
type addForm struct {
	inputs   [fieldCount]textinput.Model
	credType profile.CredentialType
	focus    addField
	spinner  spinner.Model

	submitting bool
	errMsg     string
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

func (f addForm) Update(msg tea.Msg) (addForm, tea.Cmd, *submitAddMsg) {
	if f.submitting {
		if sm, ok := msg.(spinner.TickMsg); ok {
			var cmd tea.Cmd
			f.spinner, cmd = f.spinner.Update(sm)
			return f, cmd, nil
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
		formHintStyle.Render("tab/shift+tab move · ctrl+t toggle credential type · enter next/submit · esc cancel"),
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

	if f.submitting {
		lines = append(lines, fmt.Sprintf("%s validating & saving…", f.spinner.View()))
	}
	if f.errMsg != "" {
		lines = append(lines, statusErrStyle.Render("✗ "+f.errMsg))
	}

	return appPadding.Render(strings.Join(lines, "\n"))
}
