// Package ui implements the alt-codex Bubbletea application: the dashboard,
// add-account form, and destructive-action confirmation dialog described in
// the PRD's "User Experience & Interface Design" section.
package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wfinken/alt-codex/internal/profile"
	"github.com/wfinken/alt-codex/internal/secret"
)

type view int

const (
	viewDashboard view = iota
	viewAdd
	viewConfirmDelete
)

// Model is the root Bubbletea model for alt-codex.
type Model struct {
	profiles *profile.Store
	secrets  secret.Store

	ready bool
	view  view

	items  []profile.Profile
	active string
	cursor int

	autoRefresh bool

	form    addForm
	confirm confirmDialog

	status    string
	statusErr bool

	width, height int
}

// New builds the root model. Call tea.NewProgram(New(...)).Run() to launch it.
func New(profiles *profile.Store, secrets secret.Store) Model {
	return Model{profiles: profiles, secrets: secrets, view: viewDashboard}
}

func (m Model) Init() tea.Cmd {
	return loadProfilesCmd(m.profiles)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case profilesLoadedMsg:
		m.ready = true
		if msg.err != nil {
			m.status, m.statusErr = msg.err.Error(), true
			return m, nil
		}
		m.items, m.active = msg.items, msg.active
		if m.cursor >= len(m.items) {
			if len(m.items) == 0 {
				m.cursor = 0
			} else {
				m.cursor = len(m.items) - 1
			}
		}
		return m, nil

	case switchedMsg:
		if msg.err != nil {
			m.status, m.statusErr = fmt.Sprintf("switch failed: %v", msg.err), true
			return m, clearStatusAfter(5 * time.Second)
		}
		m.status, m.statusErr = "switched to "+msg.name, false
		return m, tea.Batch(loadProfilesCmd(m.profiles), clearStatusAfter(4*time.Second))

	case addedMsg:
		if msg.err != nil {
			m.form.submitting = false
			m.form.errMsg = msg.err.Error()
			return m, nil
		}
		m.view = viewDashboard
		m.status, m.statusErr = "added profile "+msg.p.Name, false
		return m, tea.Batch(loadProfilesCmd(m.profiles), clearStatusAfter(4*time.Second))

	case deletedMsg:
		m.view = viewDashboard
		if msg.err != nil {
			m.status, m.statusErr = fmt.Sprintf("delete failed: %v", msg.err), true
			return m, clearStatusAfter(5 * time.Second)
		}
		m.status, m.statusErr = "deleted profile "+msg.name, false
		return m, tea.Batch(loadProfilesCmd(m.profiles), clearStatusAfter(4*time.Second))

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case autoRefreshTickMsg:
		if !m.autoRefresh {
			return m, nil
		}
		return m, tea.Batch(loadProfilesCmd(m.profiles), autoRefreshTick())
	}

	switch m.view {
	case viewAdd:
		return m.updateAdd(msg)
	case viewConfirmDelete:
		return m.updateConfirm(msg)
	default:
		return m.updateDashboard(msg)
	}
}

func (m Model) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch {
	case key.Matches(km, dashKeys.Quit):
		return m, tea.Quit

	case key.Matches(km, dashKeys.Up):
		if len(m.items) > 0 {
			m.cursor = (m.cursor - 1 + len(m.items)) % len(m.items)
		}

	case key.Matches(km, dashKeys.Down):
		if len(m.items) > 0 {
			m.cursor = (m.cursor + 1) % len(m.items)
		}

	case key.Matches(km, dashKeys.Add):
		m.view = viewAdd
		m.form = newAddForm()

	case key.Matches(km, dashKeys.Switch):
		if len(m.items) > 0 {
			name := m.items[m.cursor].Name
			m.status, m.statusErr = "switching to "+name+"…", false
			return m, switchCmd(m.profiles, m.secrets, name)
		}

	case key.Matches(km, dashKeys.Delete):
		if len(m.items) > 0 {
			p := m.items[m.cursor]
			msgText := fmt.Sprintf("Delete profile %q?", p.Name)
			m.confirm = newConfirmDialog(p.Name, msgText, p.Name == m.active)
			m.view = viewConfirmDelete
		}

	case key.Matches(km, dashKeys.AutoRefresh):
		m.autoRefresh = !m.autoRefresh
		if m.autoRefresh {
			m.status, m.statusErr = "auto-refresh on", false
			return m, tea.Batch(loadProfilesCmd(m.profiles), autoRefreshTick(), clearStatusAfter(2*time.Second))
		}
		m.status, m.statusErr = "auto-refresh off", false
		return m, clearStatusAfter(2 * time.Second)
	}
	return m, nil
}

func (m Model) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
		if m.form.loggingIn {
			m.form.cancelLogin()
			return m, nil
		}
		if !m.form.submitting {
			m.view = viewDashboard
			return m, nil
		}
	}

	var cmd tea.Cmd
	var sub *submitAddMsg
	m.form, cmd, sub = m.form.Update(msg)
	if sub != nil {
		return m, tea.Batch(cmd, addCmd(m.profiles, m.secrets, sub.name, sub.credType, sub.token, sub.expires))
	}
	return m, cmd
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var res *confirmResultMsg
	m.confirm, res = m.confirm.Update(msg)
	if res == nil {
		return m, nil
	}
	if !res.confirmed {
		m.view = viewDashboard
		return m, nil
	}
	wasActive := res.target == m.active
	m.status, m.statusErr = "deleting "+res.target+"…", false
	return m, deleteCmd(m.profiles, m.secrets, res.target, wasActive)
}

func (m Model) View() string {
	if !m.ready {
		return appPadding.Render("loading alt-codex…")
	}

	var body string
	switch m.view {
	case viewAdd:
		body = m.form.View()
	case viewConfirmDelete:
		body = renderDashboard(m.items, m.active, m.cursor, m.secrets.Backend(), m.autoRefresh, m.width) + "\n" + m.confirm.View()
	default:
		body = renderDashboard(m.items, m.active, m.cursor, m.secrets.Backend(), m.autoRefresh, m.width)
	}

	if m.status != "" && m.view != viewAdd {
		style := statusOKStyle
		if m.statusErr {
			style = statusErrStyle
		}
		body += "\n" + style.Render(m.status)
	}
	return body
}
