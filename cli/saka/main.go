package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	saka "github.com/sirerun/saka"
	"github.com/sirerun/saka/server"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	if cmd == "keys" {
		doKeys(args)
		return
	}

	switch cmd {
	case "search":
		doSearch(args)
	case "fetch":
		doFetch(args)
	case "serve":
		doServe(args)
	default:
		usage()
		os.Exit(1)
	}
}

func loadEngine(cfgPath string) saka.Searcher {
	cfg := saka.DefaultConfig()
	if cfgPath != "" {
		var err error
		cfg, err = saka.LoadConfig(cfgPath)
		if err != nil {
			fatal(err)
		}
	}
	engine, err := saka.New(cfg)
	if err != nil {
		fatal(err)
	}
	return engine
}

func doSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to saka.json")
	n := fs.Int("n", 10, "max results")
	format := fs.String("format", "table", "table|json|markdown")
	site := fs.String("site", "", "restrict to site")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fatal(fmt.Errorf("usage: saka search [flags] \"query\""))
	}

	e := loadEngine(*cfgPath)
	res, err := e.Search(context.Background(), saka.Query{
		Text:       strings.Join(fs.Args(), " "),
		MaxResults: *n,
		Site:       *site,
	})
	if err != nil {
		fatal(err)
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(res)
	case "markdown":
		fmt.Printf("# Results for %q _(via %s)_\n\n", res.Query, res.Provider)
		for _, r := range res.Results {
			fmt.Printf("%d. **[%s](%s)**\n   > %s\n\n", r.Position, r.Title, r.URL, r.Snippet)
		}
	default:
		for _, r := range res.Results {
			fmt.Printf("%2d. %s\n    %s\n    %s\n\n", r.Position, r.Title, r.URL, r.Snippet)
		}
	}
}

func doFetch(args []string) {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to saka.json")
	format := fs.String("format", "text", "text|json|markdown")
	fs.Parse(args)
	if fs.NArg() != 1 {
		fatal(fmt.Errorf("usage: saka fetch [flags] <url>"))
	}

	e := loadEngine(*cfgPath)
	page, err := e.Fetch(context.Background(), fs.Arg(0))
	if err != nil {
		fatal(err)
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(page)
	case "markdown":
		fmt.Printf("# %s\n\n_Source: %s_\n\n%s\n", page.Title, page.URL, page.Text)
	default:
		fmt.Println(page.Text)
	}
}

func doServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to saka.json")
	mcp := fs.Bool("mcp", false, "serve over stdio as an MCP server")
	addr := fs.String("addr", ":8080", "HTTP listen address")
	keysPath := fs.String("keys", "", "path to ed25519 public key (enables signed-key auth)")
	adminKey := fs.String("admin-key", "", "admin Bearer token for full /v1/usage dump")
	fs.Parse(args)

	e := loadEngine(*cfgPath)
	ctx := context.Background()

	if *mcp {
		if err := server.NewMCP(e).ServeStdio(ctx); err != nil {
			fatal(err)
		}
		return
	}

	opts := server.Options{AdminKey: *adminKey}
	if *keysPath != "" {
		pub, err := loadPublicKey(*keysPath)
		if err != nil {
			fatal(err)
		}
		opts.Keys = server.SignedKeys{Pub: pub}
		opts.Usage = server.NewUsageStats()
	}
	h := server.NewWithOptions(e, opts).Handler()
	fmt.Printf("saka listening on %s\n", *addr)
	fatal(http.ListenAndServe(*addr, h))
}

func loadPublicKey(path string) (ed25519.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be %d bytes, got %d", ed25519.PublicKeySize, len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "saka:", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `saka — free web search, no API keys

Usage:
  saka search "query" [-n 10] [--format table|json|markdown] [--site example.com] [--config saka.json]
  saka fetch <url> [--format text|json|markdown] [--config saka.json]
  saka serve [--addr :8080] [--mcp] [--config saka.json] [--keys saka_ed25519.pub] [--admin-key TOKEN]
  saka keys [--tier free|standard|pro] [--n 1] [--exp-days 365] [--priv path]
`)
}
