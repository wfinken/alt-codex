// Command alt-codex is a terminal UI for managing multiple Codex CLI
// authentication profiles. See PRD.md for the full product spec.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wfinken/alt-codex/internal/profile"
	"github.com/wfinken/alt-codex/internal/secret"
	"github.com/wfinken/alt-codex/internal/ui"
	"github.com/wfinken/alt-codex/internal/version"
)

func main() {
	// "current" is handled before flag.Parse() since it's a subcommand, not
	// a flag on the root TUI command (roadmap: "shell prompt integration").
	if len(os.Args) > 1 && os.Args[1] == "current" {
		os.Exit(runCurrent(os.Args[2:]))
	}

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

// runCurrent implements `alt-codex current`, a fast, non-interactive lookup
// of the active profile's name meant to be shelled out to on every prompt
// render — see the README's shell prompt integration snippets. It only ever
// reads profiles.json (never the secret store, which may hit an OS keychain
// and isn't needed here) and prints nothing with a non-zero exit when no
// profile is active — or when the dashboard's p key has turned the
// integration off — so prompt hooks hide the segment entirely instead of
// showing a blank one.
func runCurrent(args []string) int {
	fs := flag.NewFlagSet("current", flag.ExitOnError)
	showStatus := fs.Bool("status", false, "also print the active profile's health: ok, expired, or needs-reauth")
	_ = fs.Parse(args)

	profiles, err := profile.NewStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	if enabled, err := profiles.PromptIntegrationEnabled(); err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	} else if !enabled {
		return 1
	}
	items, active, err := profiles.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	if active == "" {
		return 1
	}
	if !*showStatus {
		fmt.Println(active)
		return 0
	}

	health := "ok"
	for _, p := range items {
		if p.Name != active {
			continue
		}
		if p.ExpiresAt != nil && p.ExpiresAt.Before(time.Now()) {
			if p.Type == profile.TypeRawJSON {
				health = "needs-reauth"
			} else {
				health = "expired"
			}
		}
		break
	}
	fmt.Println(active, health)
	return 0
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
