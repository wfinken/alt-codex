// Package codexrefresh silently renews a ChatGPT-OAuth profile's access
// token via the real Codex CLI's own refresh logic, without opening a
// browser — see PRD roadmap "Auto-refresh token support for profiles
// nearing expiration".
//
// alt-codex has no OAuth client of its own and does not talk to OpenAI's
// token endpoint directly. Instead, it shells out to `codex doctor` under a
// throwaway CODEX_HOME (the same sandboxing codexlogin.Start uses for full
// interactive logins) so the real Codex CLI's own refresh path does the
// work against a copy of the credential, then alt-codex reads the resulting
// auth.json back. This mirrors what `codex` already does silently for
// itself whenever it decides its own access token is stale.
//
// IMPORTANT: OpenAI's OAuth backend rotates the refresh_token on every use.
// That means the JSON Try returns must be persisted back to the profile's
// secret store whenever err is nil — even when Try reports refreshed=false
// — because the attempt itself may have invalidated the refresh_token the
// caller started with. Discarding the result on a "no-op" leaves the
// profile holding a dead refresh_token.
package codexrefresh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NearExpiryWindow is how far ahead of a token's real expiry alt-codex
// starts attempting silent renewal.
const NearExpiryWindow = 24 * time.Hour

// refreshTimeout bounds how long a single silent-refresh attempt may take.
const refreshTimeout = 20 * time.Second

// ExpiryOf reports the expiry time encoded in the ChatGPT access token
// embedded in authJSON (a Codex CLI auth.json document), decoded from the
// token's own JWT `exp` claim — no network call, no signature verification
// (alt-codex only needs the claim's local bookkeeping value, not to
// authenticate it; the real server-side check happens when the token is
// actually used). ok is false for anything that isn't a parseable
// ChatGPT-OAuth token, such as a bare API-key profile.
func ExpiryOf(authJSON string) (t time.Time, ok bool) {
	var doc struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal([]byte(authJSON), &doc); err != nil {
		return time.Time{}, false
	}
	parts := strings.Split(doc.Tokens.AccessToken, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}

// Try attempts a silent, non-interactive renewal of authJSON's access token
// using its embedded refresh_token, by running `codex doctor` against a
// throwaway CODEX_HOME seeded with a copy of authJSON.
//
// newJSON is the (possibly rotated) resulting auth.json and must be
// persisted by the caller whenever err is nil, regardless of refreshed —
// see the package doc comment. refreshed is true only if the access
// token's expiry actually moved forward, meaning the refresh_token was
// still good. refreshed=false with err=nil means the refresh_token itself
// is dead and an interactive `codex login` is the only way forward.
func Try(authJSON string) (newJSON string, refreshed bool, err error) {
	if _, err := exec.LookPath("codex"); err != nil {
		return "", false, fmt.Errorf("codex CLI not found on PATH: %w", err)
	}

	beforeExp, hadExp := ExpiryOf(authJSON)

	tempHome, err := os.MkdirTemp("", "alt-codex-refresh-")
	if err != nil {
		return "", false, err
	}
	defer func() { _ = os.RemoveAll(tempHome) }()

	authPath := filepath.Join(tempHome, "auth.json")
	if err := os.WriteFile(authPath, []byte(authJSON), 0o600); err != nil {
		return "", false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "codex", "doctor", "--json")
	cmd.Env = append(os.Environ(), "CODEX_HOME="+tempHome)
	// `codex doctor` reports on many unrelated checks (desktop app,
	// git environment, etc.) and can exit non-zero for reasons that have
	// nothing to do with auth, so its exit status alone isn't a reliable
	// signal here — the auth.json it leaves behind is.
	_ = cmd.Run()

	data, err := os.ReadFile(authPath)
	if err != nil {
		return "", false, fmt.Errorf("read refreshed auth.json: %w", err)
	}
	newJSON = string(data)

	afterExp, ok := ExpiryOf(newJSON)
	refreshed = hadExp && ok && afterExp.After(beforeExp)
	return newJSON, refreshed, nil
}
