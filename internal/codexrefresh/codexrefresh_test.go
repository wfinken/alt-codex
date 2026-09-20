package codexrefresh

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func fakeJWT(t *testing.T, exp int64) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, err := json.Marshal(map[string]any{"exp": exp})
	if err != nil {
		t.Fatal(err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func authJSON(t *testing.T, accessToken string) string {
	t.Helper()
	doc := map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]any{
			"access_token":  accessToken,
			"refresh_token": "rt",
		},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExpiryOfParsesJWTExpClaim(t *testing.T) {
	want := time.Now().Add(2 * time.Hour).Unix()
	got, ok := ExpiryOf(authJSON(t, fakeJWT(t, want)))
	if !ok {
		t.Fatal("ExpiryOf reported ok=false for a well-formed token")
	}
	if got.Unix() != want {
		t.Errorf("got exp %v, want %v", got.Unix(), want)
	}
}

func TestExpiryOfRejectsNonJWTCredential(t *testing.T) {
	if _, ok := ExpiryOf(`{"OPENAI_API_KEY":"sk-not-a-jwt-at-all"}`); ok {
		t.Error("ExpiryOf reported ok=true for a bare API-key profile")
	}
}

func TestExpiryOfRejectsMalformedJSON(t *testing.T) {
	if _, ok := ExpiryOf("not json"); ok {
		t.Error("ExpiryOf reported ok=true for malformed JSON")
	}
}
