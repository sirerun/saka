package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	if err := runSearch(loadEngine, args, os.Stdout); err != nil {
		fatal(err)
	}
}

// runSearch holds doSearch's logic behind a testable seam: newEngine builds
// the Searcher from the parsed --config flag, and output goes to stdout
// instead of directly to os.Stdout, so tests can inject a fake Searcher and
// capture output.
func runSearch(newEngine func(cfgPath string) saka.Searcher, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to saka.json")
	n := fs.Int("n", 10, "max results")
	format := fs.String("format", "table", "table|json|markdown")
	site := fs.String("site", "", "restrict to site")
	vertical := fs.String("vertical", "", "search vertical (e.g. news); empty = general web")
	stream := fs.Bool("stream", false, "print results incrementally as they arrive via SearchStream instead of waiting for the full batch")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: saka search [flags] \"query\"")
	}

	e := newEngine(*cfgPath)
	q := saka.Query{
		Text:       strings.Join(fs.Args(), " "),
		MaxResults: *n,
		Site:       *site,
		Vertical:   *vertical,
	}

	if *stream {
		return streamSearch(e, q, *format, stdout)
	}

	res, err := e.Search(context.Background(), q)
	if err != nil {
		return err
	}
	printResults(stdout, res, *format)
	return nil
}

// streamSearch drains e.SearchStream's item channel, printing each Result as
// it arrives, then reads the done/error channel for the final outcome.
func streamSearch(e saka.Searcher, q saka.Query, format string, stdout io.Writer) error {
	itemCh, doneCh, errCh := e.SearchStream(context.Background(), q)
	for r := range itemCh {
		printResult(stdout, r, format)
	}
	select {
	case <-doneCh:
		return nil
	case err := <-errCh:
		return err
	}
}

func printResults(w io.Writer, res *saka.Results, format string) {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	case "markdown":
		_, _ = fmt.Fprintf(w, "# Results for %q _(via %s)_\n\n", res.Query, res.Provider)
		for _, r := range res.Results {
			printResult(w, r, format)
		}
	default:
		for _, r := range res.Results {
			printResult(w, r, format)
		}
	}
}

func printResult(w io.Writer, r saka.Result, format string) {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		_ = enc.Encode(r)
	case "markdown":
		_, _ = fmt.Fprintf(w, "%d. **[%s](%s)**\n   > %s\n\n", r.Position, r.Title, r.URL, r.Snippet)
	default:
		_, _ = fmt.Fprintf(w, "%2d. %s\n    %s\n    %s\n\n", r.Position, r.Title, r.URL, r.Snippet)
	}
}

func doFetch(args []string) {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to saka.json")
	format := fs.String("format", "text", "text|json|markdown")
	_ = fs.Parse(args)
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
		_ = enc.Encode(page)
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
	_ = fs.Parse(args)

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
  saka search "query" [-n 10] [--format table|json|markdown] [--site example.com] [--vertical news] [--stream] [--config saka.json]
  saka fetch <url> [--format text|json|markdown] [--config saka.json]
  saka serve [--addr :8080] [--mcp] [--config saka.json] [--keys saka_ed25519.pub] [--admin-key TOKEN]
  saka keys [--tier free|standard|pro] [--n 1] [--exp-days 365] [--priv path]
`)
}
