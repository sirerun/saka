// Package server — Ed25519-signed API keys, verified offline (no DB lookup
// needed at request time).
//
// Key format:
//
//	sk-<version>.<base64url(payload)>.<base64url(sig)>
//
// payload (JSON, then base64url-encoded):
//
//	{ "id": "k_7f3a", "tier": "pro", "exp": "2026-01-01T00:00:00Z", "n": 1 }
package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type KeyPayload struct {
	ID   string    `json:"id"`
	Tier string    `json:"tier"`
	Exp  time.Time `json:"exp"`
	N    int       `json:"n"` // rotation counter
}

// GenerateKeyPair creates an ed25519 signing pair. Store the private key
// securely; ship the public key with the server for offline verification.
func GenerateKeyPair() (pub ed25519.PublicKey, priv ed25519.PrivateKey, err error) {
	pub, priv, err = ed25519.GenerateKey(rand.Reader)
	return
}

func SignKey(priv ed25519.PrivateKey, p KeyPayload) (string, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	b64 := base64.RawURLEncoding.EncodeToString
	sig := ed25519.Sign(priv, body)
	return fmt.Sprintf("sk-1.%s.%s", b64(body), b64(sig)), nil
}

// VerifyKey checks the signature and expiry of a key produced by SignKey.
//
// NOTE (source-chat bugs, fixed): the chat's version called an undefined
// `splitN(...)` (with a comment "manual: strings.Split, len check" as if
// unsure whether to inline it) and had a bare `return p,` missing the
// error value on the base64-decode-failure path. Both fixed below.
func VerifyKey(pub ed25519.PublicKey, key string) (KeyPayload, error) {
	var p KeyPayload
	parts := strings.SplitN(key, ".", 3)
	if len(parts) != 3 || parts[0] != "sk-1" {
		return p, fmt.Errorf("malformed key")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return p, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return p, err
	}
	if !ed25519.Verify(pub, body, sig) {
		return p, fmt.Errorf("invalid signature")
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return p, err
	}
	if !p.Exp.IsZero() && time.Now().After(p.Exp) {
		return p, fmt.Errorf("key expired")
	}
	return p, nil
}

// SignedKeys verifies signed keys offline. Falls back to an optional
// revocation list for revoking individual key IDs.
type SignedKeys struct {
	Pub     ed25519.PublicKey
	Revoked map[string]bool // key ID -> revoked
}

func (s SignedKeys) Lookup(key string) (string, bool) {
	p, err := VerifyKey(s.Pub, key)
	if err != nil || s.Revoked[p.ID] {
		return "", false
	}
	return p.Tier, true
}
