// Command alt-codex is a terminal UI for managing multiple Codex CLI
// authentication profiles. See PRD.md for the full product spec.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	xterm "github.com/charmbracelet/x/term"
	"github.com/wfinken/alt-codex/internal/backup"
	"github.com/wfinken/alt-codex/internal/profile"
	"github.com/wfinken/alt-codex/internal/secret"
	"github.com/wfinken/alt-codex/internal/ui"
	"github.com/wfinken/alt-codex/internal/version"
)

func main() {
	// These are handled before flag.Parse() since they're subcommands, not
	// flags on the root TUI command (roadmap: "shell prompt integration",
	// "encrypted import/export").
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "current":
			os.Exit(runCurrent(os.Args[2:]))
		case "export":
			os.Exit(runExport(os.Args[2:]))
		case "import":
			os.Exit(runImport(os.Args[2:]))
		}
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

// runExport implements `alt-codex export <file>` (roadmap: "encrypted
// import/export"). It bundles every profile's metadata and secret into one
// passphrase-encrypted archive suitable for backup or moving to another
// machine — see internal/backup for the file format.
func runExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: alt-codex export <file>")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	path := fs.Arg(0)

	profiles, err := profile.NewStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	items, _, err := profiles.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "alt-codex: no profiles to export")
		return 1
	}

	secrets, err := secret.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	arc := backup.Archive{
		Version:    backup.FormatVersion,
		ExportedAt: time.Now().UTC(),
		Profiles:   items,
		Secrets:    map[string]string{},
	}
	for _, p := range items {
		v, err := secrets.Get(p.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "alt-codex: skipping %s, no secret found: %v\n", p.Name, err)
			continue
		}
		arc.Secrets[p.Name] = v
	}

	passphrase, err := readNewPassphrase()
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	data, err := backup.Encrypt(arc, passphrase)
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}

	fmt.Printf("Exported %d profile(s) to %s\n", len(arc.Profiles), path)
	return 0
}

// runImport implements `alt-codex import <file>`, the restore side of
// encrypted import/export. Profiles already present in the local store are
// left untouched unless --overwrite is given, so restoring a backup can
// never silently clobber newer local work.
func runImport(args []string) int {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	overwrite := fs.Bool("overwrite", false, "replace existing profiles that share a name with one being imported")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: alt-codex import [--overwrite] <file>")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	passphrase, err := readPassphrase("Import passphrase: ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	arc, err := backup.Decrypt(data, passphrase)
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}

	profiles, err := profile.NewStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}
	res, err := profiles.Import(arc.Profiles, *overwrite)
	if err != nil {
		fmt.Fprintln(os.Stderr, "alt-codex:", err)
		return 1
	}

	if len(res.Imported) > 0 {
		secrets, err := secret.New()
		if err != nil {
			fmt.Fprintln(os.Stderr, "alt-codex:", err)
			return 1
		}
		for _, name := range res.Imported {
			v, ok := arc.Secrets[name]
			if !ok {
				continue
			}
			if err := secrets.Set(name, v); err != nil {
				fmt.Fprintf(os.Stderr, "alt-codex: store secret for %s: %v\n", name, err)
			}
		}
	}

	fmt.Printf("Imported %d profile(s)", len(res.Imported))
	if len(res.Skipped) > 0 {
		fmt.Printf(", skipped %d already present (use --overwrite to replace): %s", len(res.Skipped), strings.Join(res.Skipped, ", "))
	}
	fmt.Println()
	return 0
}

// readNewPassphrase prompts twice so a typo in a passphrase the user will
// never see again (it's never stored) doesn't silently lock them out of
// their own backup.
func readNewPassphrase() (string, error) {
	p1, err := readPassphrase("Export passphrase: ")
	if err != nil {
		return "", err
	}
	if p1 == "" {
		return "", fmt.Errorf("passphrase must not be empty")
	}
	p2, err := readPassphrase("Confirm passphrase: ")
	if err != nil {
		return "", err
	}
	if p1 != p2 {
		return "", fmt.Errorf("passphrases did not match")
	}
	return p1, nil
}

// readPassphrase reads one passphrase, masked when stdin is a real terminal.
// ALT_CODEX_PASSPHRASE lets scripts and tests supply it non-interactively,
// same idea as e.g. AWS_* or GPG_* env overrides for otherwise-prompted CLIs.
func readPassphrase(prompt string) (string, error) {
	if v := os.Getenv("ALT_CODEX_PASSPHRASE"); v != "" {
		return v, nil
	}

	fmt.Fprint(os.Stderr, prompt)
	fd := os.Stdin.Fd()
	if xterm.IsTerminal(fd) {
		b, err := xterm.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
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
