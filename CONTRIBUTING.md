# Contributing to alt-codex

Thanks for taking the time to contribute! 🎉

## Getting started

```sh
git clone https://github.com/wfinken/alt-codex.git
cd alt-codex
make run
```

Requires Go 1.23+.

## Before opening a PR

```sh
make fmt
make lint
make test
go build ./...
```

## Project layout

```
cmd/alt-codex/       entrypoint
internal/profile/    profile metadata (profiles.json)
internal/secret/      OS keychain + encrypted-file secret storage
internal/codexconfig/ applies the active profile to the Codex CLI's auth file
internal/validate/    input validation (FR-07)
internal/ui/           the Bubbletea TUI
```

## Reporting bugs / requesting features

Please use the issue templates — they help us get the info we need on the first pass.

## A note on secrets

Never commit real tokens, `profiles.json`, or `config.enc` in a PR (the `.gitignore`
already excludes the ones alt-codex generates at runtime). Test with throwaway
credentials only.
