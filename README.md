# ⌁ alt-codex

**Switch Codex accounts at the speed of thought.**

`alt-codex` is a fast, keyboard-driven terminal UI for juggling multiple
[Codex CLI](https://openai.com) accounts — work, personal, side-project,
whatever — without ever hand-editing a config file or copy-pasting a token
again.

[![CI](https://github.com/wfinken/alt-codex/actions/workflows/ci.yml/badge.svg)](https://github.com/wfinken/alt-codex/actions/workflows/ci.yml)
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

  ▸ work-corp                 ● Active    last switched 2026-09-20 08:14
    personal-github           ○ Saved
    side-quest-startup        ⟳ Needs re-auth
    old-consulting-gig        ✕ Expired

  options: OS keychain + encrypted-file fallback · auto-refresh off · renew mode ask · shell prompt on
  ↑/k ↓/j navigate   enter/s switch   a add   d delete   ? settings & shortcuts   q quit
```

## Features

- ⚡ **Instant switching** — select a profile, hit `enter`, done. No shell restarts.
- 🔐 **Secure by default** — credentials live in your OS keychain (macOS
  Keychain, Linux Secret Service, Windows Credential Manager), never in
  plaintext. Falls back to an AES-256-GCM encrypted local file automatically
  whenever no keychain is available *or* a secret is too large for one (OS
  keychains cap item size, and a full ChatGPT OAuth credential can exceed it).
- 🧭 **Full keyboard control** — arrow keys or vim motions (`j`/`k`), your call.
- ➕ **Guided onboarding** — an in-TUI form validates a new token before it's
  ever saved.
- 🔑 **Sign in with ChatGPT** — no API key? No problem. `ctrl+l` in the add
  form runs the real Codex CLI's own browser-based OAuth login for you and
  captures the resulting credential automatically — perfect for work
  accounts that only have a ChatGPT/Codex seat, not API access.
- 🗑️ **Safe deletion** — deleting your *active* profile prompts an extra
  confirmation so you don't lock yourself out mid-task.
- ↻ **Auto-refresh** — press `r` on the dashboard to have alt-codex reload
  profiles every 30s on its own, so a badge shows up the moment a profile
  changes state instead of waiting for your next action.
- 🔄 **Auto-renew ChatGPT tokens** — alt-codex silently renews a
  ChatGPT-OAuth profile's access token via the real Codex CLI's own refresh
  logic (no browser, no prompt) whenever it's within 24h of expiring —
  checked on startup and on every auto-refresh tick. If a profile's
  refresh_token itself has died and only a full re-login can fix it, what
  happens next depends on the renew mode (cycle with `m`): **ask** (default)
  shows a hint naming the profile so you press `l` to sign in; **auto**
  pops the ChatGPT OAuth page in your browser immediately, no prompt;
  **manual** just flags the `⟳ Needs re-auth` badge and waits for you to
  press `l` whenever you're ready. `l` re-authenticates the selected
  profile in place any time, in any mode.
- 🖥️ **Cross-platform** — macOS, Linux, and Windows.

## Install

### Homebrew (macOS/Linux)

```sh
brew install wfinken/tap/alt-codex
```

### APT (Debian/Ubuntu)

```sh
curl -fsSL https://wfinken.github.io/alt-codex/apt/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/alt-codex.gpg
echo "deb [signed-by=/usr/share/keyrings/alt-codex.gpg] https://wfinken.github.io/alt-codex/apt/ /" | sudo tee /etc/apt/sources.list.d/alt-codex.list
sudo apt update
sudo apt install alt-codex
```

### Go

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

Requires Go 1.24.2+ to build. No runtime dependencies once compiled.

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
| `r`           | Toggle auto-refresh              |
| `l`           | Sign in / re-authenticate the selected profile |
| `m`           | Cycle renew mode (ask/auto/manual) |
| `p`           | Toggle shell prompt integration (on by default) |
| `?`           | Open settings & shortcuts        |
| `q`           | Quit                              |
| `esc`         | Back / cancel                    |

The dashboard itself only shows the essentials — active profile, the list,
and a muted `options:` summary line. Press `?` any time for the full
shortcut legend plus each setting's current value (auto-refresh, renew mode,
shell prompt integration, secrets backend) in one place; the single-key
toggles above still work directly from the dashboard without opening it.

`p` is persisted to `profiles.json`, not just the current session, since
[`alt-codex current`](#shell-prompt-integration) runs as its own process from
your shell — turning the integration off here makes it stop reporting a
profile (same as having none active) until you turn it back on.

When adding a profile, `tab`/`shift+tab` move between fields, `ctrl+t`
toggles between a bare API key and a full pasted `auth.json` payload, and
`ctrl+l` signs in with the real Codex CLI (requires `codex` on your `PATH`)
and fills the credential in for you — no copy-paste required. It runs under
an isolated, throwaway `CODEX_HOME` so it never disturbs whatever account
your normal `codex` commands are currently using.

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

## Shell prompt integration

`alt-codex current` prints just the active profile's name — no TUI, no
keychain access, cheap enough to shell out to on every prompt render. It
exits non-zero with no output when no profile is active, or when the
integration is turned off (`p` in the dashboard), so the hooks below hide
the segment entirely instead of showing a blank one.

```sh
$ alt-codex current
work-corp
```

Add `--status` to also get a health word (`ok`, `expired`, or
`needs-reauth`) for coloring the segment when a token needs attention:

```sh
$ alt-codex current --status
work-corp ok
```

**Bash** (`~/.bashrc`):
```sh
alt_codex_ps1() {
  local p; p=$(alt-codex current 2>/dev/null) || return
  printf '⌁ %s ' "$p"
}
PS1='$(alt_codex_ps1)'"$PS1"
```

**Zsh** (`~/.zshrc`):
```sh
alt_codex_prompt() {
  local p; p=$(alt-codex current 2>/dev/null) || return
  echo "⌁ %F{magenta}${p}%f "
}
setopt PROMPT_SUBST
PROMPT='$(alt_codex_prompt)'"$PROMPT"
```

**Fish** (`~/.config/fish/functions/fish_prompt.fish`):
```fish
function fish_prompt
    set -l p (alt-codex current 2>/dev/null)
    if test -n "$p"
        set_color magenta
        echo -n "⌁$p "
        set_color normal
    end
    # ...rest of your prompt
end
```

**Starship** (`~/.config/starship.toml`):
```toml
[custom.alt_codex]
command = "alt-codex current"
when = "alt-codex current >/dev/null 2>&1"
format = "[⌁ $output]($style) "
style = "bold purple"
```

To flip the segment red when a profile needs re-auth, swap in the
`--status` form and branch on the second word — e.g. in the bash/zsh
functions above:

```sh
alt_codex_prompt() {
  local out p health; out=$(alt-codex current --status 2>/dev/null) || return
  p=${out%% *}; health=${out#* }
  local color=magenta; [ "$health" != ok ] && color=red
  echo "⌁ %F{$color}${p}%f "
}
```

## Roadmap

- [x] Auto-refresh tokens nearing expiration (silent renewal + `l`/`m` re-auth controls; `r` toggles dashboard auto-refresh)
- [x] Shell prompt integration (`alt-codex current` + bash/zsh/fish/starship snippets above)
- [ ] Encrypted import/export for backing up profiles

## Contributing

PRs welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © William Finken
