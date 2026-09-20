// Package version holds build-time metadata injected via -ldflags.
package version

// Version is overridden at build time, e.g.:
//
//	go build -ldflags "-X github.com/wfinken/alt-codex/internal/version.Version=v0.1.0"
var Version = "dev"
