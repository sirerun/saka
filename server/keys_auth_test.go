package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSignedKeysLookup(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	key, err := SignKey(priv, KeyPayload{
		ID:   "k_test",
		Tier: "pro",
		Exp:  time.Now().Add(time.Hour),
		N:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	sk := SignedKeys{Pub: pub}
	tier, ok := sk.Lookup(key)
	if !ok || tier != "pro" {
		t.Fatalf("tier=%q ok=%v", tier, ok)
	}
	if _, ok := sk.Lookup("sk-1.bad.bad"); ok {
		t.Fatal("expected invalid")
	}
}

func TestSignedKeysRevoked(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	key, _ := SignKey(priv, KeyPayload{ID: "k_rev", Tier: "free", Exp: time.Now().Add(time.Hour)})
	sk := SignedKeys{Pub: pub, Revoked: map[string]bool{"k_rev": true}}
	if _, ok := sk.Lookup(key); ok {
		t.Fatal("revoked key accepted")
	}
}

func TestAuthMiddlewareRequiresKey(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	key, _ := SignKey(priv, KeyPayload{ID: "k1", Tier: "standard", Exp: time.Now().Add(time.Hour)})
	h := AuthMiddleware(SignedKeys{Pub: pub}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if APIKeyFromContext(r.Context()) != key {
			t.Errorf("context key mismatch")
		}
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/search", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/search", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d %s", rr.Code, rr.Body.String())
	}
}

func TestVerifyKeyExpired(t *testing.T) {
	pub, priv, _ := GenerateKeyPair()
	key, _ := SignKey(priv, KeyPayload{ID: "k", Tier: "free", Exp: time.Now().Add(-time.Hour)})
	_, err := VerifyKey(pub, key)
	if err == nil {
		t.Fatal("expected expiry error")
	}
}
