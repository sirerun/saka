package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirerun/saka/server"
)

func doKeys(args []string) {
	fs := flag.NewFlagSet("keys", flag.ExitOnError)
	tier := fs.String("tier", "free", "free|standard|pro")
	privPath := fs.String("priv", "saka_ed25519.key", "private key file (created on first run)")
	expDays := fs.Int("exp-days", 365, "days until expiry (0 = never)")
	count := fs.Int("n", 1, "number of keys to generate")
	_ = fs.Parse(args)

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
		priv := ed25519.PrivateKey(key)
		writePubBeside(path, priv)
		return priv
	}
	_, priv, err := server.GenerateKeyPair()
	fatalIf(err)
	fatalIf(os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600))
	writePubBeside(path, priv)
	fmt.Fprintf(os.Stderr, "generated new signing key at %s — back it up!\n", path)
	return priv
}

func writePubBeside(privPath string, priv ed25519.PrivateKey) {
	pub := priv.Public().(ed25519.PublicKey)
	pubPath := strings.TrimSuffix(privPath, filepath.Ext(privPath)) + ".pub"
	if filepath.Ext(privPath) == "" {
		pubPath = privPath + ".pub"
	}
	_ = os.WriteFile(pubPath, []byte(base64.StdEncoding.EncodeToString(pub)), 0o644)
	fmt.Fprintf(os.Stderr, "public key: %s (pass to saka serve --keys)\n", pubPath)
}

func fatalIf(err error) {
	if err != nil {
		fatal(err)
	}
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
