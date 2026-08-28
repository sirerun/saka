package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/you/saka/server"
)

// doKeys implements `saka keys` — generates Ed25519-signed API keys.
//
// NOTE (source-chat bugs, fixed): the chat's version called
// `loadOrCreatePrivprivPath)` (missing the `(` — a dropped character) and
// built the KeyPayload literal as
//
//	p := KeyPayload{
//	    :   fmt.Sprintf("k_%s randHex(4)),
//	    Tier: *tier,
//	    N:    1,
//	}
//
// which is missing the `ID:` field name, the comma/format-string close
// around `randHex(4)`, and doesn't compile. Reconstructed to evident
// intent below.
func doKeys(args []string) {
	fs := flag.NewFlagSet("keys", flag.ExitOnError)
	tier := fs.String("tier", "free", "free|standard|pro")
	privPath := fs.String("priv", "saka_ed25519.key", "private key file (created on first run)")
	expDays := fs.Int("exp-days", 365, "days until expiry (0 = never)")
	count := fs.Int("n", 1, "number of keys to generate")
	fs.Parse(args)

	priv := loadOrCreatePriv(*privPath)

	for i := 0; i < *count; i++ {
		p := server.KeyPayload{
			ID:   fmt.Sprintf("k_%s", randHex(4)),
			Tier: *tier,
			N:    1,
		}
		if *expDays > 0 {
			p.Exp = time.Now().AddDate(0, 0, *expDays)
		}
		key, err := server.SignKey(priv, p)
		fatalIf(err)
		fmt.Printf("%s\n", key)
	}
}

func loadOrCreatePriv(path string) ed25519.PrivateKey {
	if b, err := os.ReadFile(path); err == nil {
		key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
		fatalIf(err)
		return ed25519.PrivateKey(key)
	}
	_, priv, err := server.GenerateKeyPair()
	fatalIf(err)
	os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600)
	fmt.Fprintf(os.Stderr, "generated new signing key at %s — back it up!\n", path)
	return priv
}

func fatalIf(err error) {
	if err != nil {
		fatal(err)
	}
}

// randHex returns n random bytes hex-encoded. Referenced by doKeys in the
// source chat but never defined there — added so this file compiles.
func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Usage:
//   saka keys --tier pro --n 5 --exp-days 30
//   # sk-1.eyJpZCI6ImtfxxxxIsInRpZXIiOiJwcm8i...  (×5)
