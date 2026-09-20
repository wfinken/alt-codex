// Command alt-codex is a terminal UI for managing multiple Codex CLI
// authentication profiles. See PRD.md for the full product spec.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wfinken/alt-codex/internal/profile"
	"github.com/wfinken/alt-codex/internal/secret"
	"github.com/wfinken/alt-codex/internal/ui"
	"github.com/wfinken/alt-codex/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print alt-codex's version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("alt-codex " + version.Version)
		return
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		os.Exit(1)
	}
}

func run() error {
	profiles, err := profile.NewStore()
	if err != nil {
		return fmt.Errorf("open profile store: %w", err)
	}

	secrets, err := secret.New()
	if err != nil {
		return fmt.Errorf("open secret store: %w", err)
	}

	p := tea.NewProgram(ui.New(profiles, secrets), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
