package ui

import "github.com/charmbracelet/bubbles/key"

type dashboardKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Switch      key.Binding
	Add         key.Binding
	Delete      key.Binding
	AutoRefresh key.Binding
	Reauth      key.Binding
	RenewMode   key.Binding
	Help        key.Binding
	Quit        key.Binding
}

var dashKeys = dashboardKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Switch: key.NewBinding(
		key.WithKeys("enter", "s"),
		key.WithHelp("enter/s", "switch"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	AutoRefresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "auto-refresh"),
	),
	Reauth: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "sign in"),
	),
	RenewMode: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "renew mode"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "settings & shortcuts"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
