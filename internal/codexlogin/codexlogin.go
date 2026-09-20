// Package codexlogin drives the real Codex CLI's own `codex login` flow so
// alt-codex can onboard ChatGPT-OAuth accounts (no API key) without the
// user manually copying auth.json by hand.
//
// It intentionally uses the standard browser-redirect flow rather than
// `--device-auth`: alt-codex always runs locally with a real browser
// available, and many enterprise identity providers specifically disable
// the OAuth device-code grant (it's a known phishing vector) while still
// allowing normal browser-redirect OAuth — so device-auth is more likely
// to be blocked by a work account's org policy for no benefit here.
//
// The login runs under a throwaway CODEX_HOME so it can never disturb
// whichever profile is currently active for the user's normal `codex`
// invocations; only the resulting auth.json bytes are pulled out before the
// temp directory is destroyed.
package codexlogin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Session tracks one in-flight (or finished) login attempt. It is safe to
// poll concurrently from a UI event loop via Snapshot.
type Session struct {
	mu       sync.Mutex
	lines    []string
	done     bool
	err      error
	authJSON string
	cancel   context.CancelFunc
}

// Start launches `codex login` in the background and returns immediately;
// poll Snapshot to observe progress and completion.
func Start() *Session {
	s := &Session{}

	if _, err := exec.LookPath("codex"); err != nil {
		s.finish("", errors.New("codex CLI not found on PATH — install it first, then try again"))
		return s
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.run(ctx)
	return s
}

// Cancel aborts an in-flight login attempt. Safe to call multiple times.
func (s *Session) Cancel() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Snapshot returns the output captured so far, whether the session has
// finished, and — once finished — either an error or the captured
// auth.json contents.
func (s *Session) Snapshot() (lines []string, done bool, authJSON string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.lines...), s.done, s.authJSON, s.err
}

func (s *Session) run(ctx context.Context) {
	tempHome, err := os.MkdirTemp("", "alt-codex-login-")
	if err != nil {
		s.finish("", err)
		return
	}
	defer func() { _ = os.RemoveAll(tempHome) }()

	cmd := exec.CommandContext(ctx, "codex", "login")
	cmd.Env = append(os.Environ(), "CODEX_HOME="+tempHome)
	out := &lineWriter{s: s}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Start(); err != nil {
		s.finish("", fmt.Errorf("couldn't start codex login: %w", err))
		return
	}
	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.Canceled {
			s.finish("", errors.New("login cancelled"))
			return
		}
		s.finish("", fmt.Errorf("codex login failed: %w", err))
		return
	}

	data, err := os.ReadFile(filepath.Join(tempHome, "auth.json"))
	if err != nil {
		s.finish("", errors.New("codex login succeeded but wrote no auth.json — your Codex CLI may be "+
			`configured with cli_auth_credentials_store = "keyring"; set it to "file" in ~/.codex/config.toml `+
			"and try again, or paste the credential manually"))
		return
	}
	s.finish(string(data), nil)
}

func (s *Session) appendLine(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines = append(s.lines, line)
}

func (s *Session) finish(authJSON string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done = true
	s.err = err
	s.authJSON = authJSON
}

// lineWriter splits a subprocess's combined stdout/stderr into lines as
// they arrive, so the UI can show login progress as it's printed rather
// than only once the process exits.
type lineWriter struct {
	s   *Session
	buf []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:i]), "\r")
		if line != "" {
			w.s.appendLine(line)
		}
		w.buf = w.buf[i+1:]
	}
	return len(p), nil
}
