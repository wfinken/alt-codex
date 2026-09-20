# ⌁ alt-codex

**Switch Codex accounts at the speed of thought.**

`alt-codex` is a fast, keyboard-driven terminal UI for juggling multiple
[Codex CLI](https://openai.com) accounts — work, personal, side-project,
whatever — without ever hand-editing a config file or copy-pasting a token
again.

[![CI](https://github.com/wfinken/alt-codex/actions/workflows/ci.yml/badge.svg)](https://github.com/wfinken/alt-codex/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wfinken/alt-codex)](https://goreportcard.com/report/github.com/wfinken/alt-codex)
[![Go Reference](https://pkg.go.dev/badge/github.com/wfinken/alt-codex.svg)](https://pkg.go.dev/github.com/wfinken/alt-codex)
[![Go version](https://img.shields.io/github/go-mod/go-version/wfinken/alt-codex)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Built with Bubbletea](https://img.shields.io/badge/built%20with-%F0%9F%AB%A7%20bubbletea-ff69b4)](https://github.com/charmbracelet/bubbletea)

---

## Why?

If you've ever run `codex` on the wrong account, or dug through
`~/.codex/auth.json` trying to remember which token was which — `alt-codex`
is for you. It's a tiny, dependency-light TUI that sits on top of the Codex
CLI and gives every account a name, a badge, and a single keystroke.

```
  ⌁ alt-codex    active: work-corp
  secrets backed by: OS keychain

  ▸ work-corp                 ● Active    last switched 2026-09-20 08:14
    personal-github           ○ Saved
    side-quest-startup        ○ Saved
    old-consulting-gig        ✕ Expired

  ↑/k ↓/j navigate   enter/s switch   a add   d delete   q quit
```

## Features

- ⚡ **Instant switching** — select a profile, hit `enter`, done. No shell restarts.
- 🔐 **Secure by default** — credentials live in your OS keychain (macOS
  Keychain, Linux Secret Service, Windows Credential Manager), never in
  plaintext, with an AES-256-GCM encrypted local fallback if no keychain is
  available.
- 🧭 **Full keyboard control** — arrow keys or vim motions (`j`/`k`), your call.
- ➕ **Guided onboarding** — an in-TUI form validates a new token before it's
  ever saved.
- 🗑️ **Safe deletion** — deleting your *active* profile prompts an extra
  confirmation so you don't lock yourself out mid-task.
- 🖥️ **Cross-platform** — macOS, Linux, and Windows.

## Install

```sh
go install github.com/wfinken/alt-codex/cmd/alt-codex@latest
```

Or build from source:

```sh
git clone https://github.com/wfinken/alt-codex.git
cd alt-codex
make build
./alt-codex
```

Requires Go 1.23+ to build. No runtime dependencies once compiled.

## Usage

Just run:

```sh
alt-codex
```

| Key           | Action                          |
| ------------- | -------------------------------- |
| `↑`/`k` `↓`/`j` | Move the selection               |
| `enter` / `s` | Switch to the selected profile   |
| `a`           | Add a new account                |
| `d`           | Delete the selected profile      |
| `q`           | Quit                              |
| `esc`         | Back / cancel                    |

When adding a profile, `tab`/`shift+tab` move between fields and `ctrl+t`
toggles between a bare API key and a full pasted `auth.json` payload.

## How it works

```
┌─────────────┐        ┌──────────────────────┐        ┌───────────────────┐
│  alt-codex  │──────▶│ ~/.config/alt-codex/  │        │   OS Keychain /    │
│    (TUI)    │        │      profiles.json    │        │  encrypted fallback│
└─────┬───────┘        │   (names + metadata,  │        │  (actual secrets)  │
      │                │      no secrets)       │◀──────┤                    │
      │                └──────────────────────┘        └───────────────────┘
      │ on switch
      ▼
┌─────────────────────┐
│  ~/.codex/auth.json   │   ← what the real Codex CLI reads
└─────────────────────┘
```

Profile *metadata* (names, timestamps, credential type) lives in
`~/.config/alt-codex/profiles.json`. The *secrets* themselves never touch
that file — they're stored via your OS keychain, or in an encrypted
`config.enc` fallback if no keychain is reachable. Switching profiles writes
the selected credential straight into the Codex CLI's own auth file, so
Codex picks it up immediately.

## Roadmap

- [ ] Auto-refresh tokens nearing expiration
- [ ] Shell prompt integration (show the active profile in your prompt)
- [ ] Encrypted import/export for backing up profiles

## Contributing

PRs welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © William Finken
