package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	saka "github.com/you/saka"
	"github.com/you/saka/server"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var cfgPath string
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&cfgPath, "config", "", "path to saka.json")

	cmd := os.Args[1]
	fs.Parse(os.Args[2:]) // consume global flags if present

	// `keys` doesn't need an engine — handle it before constructing one.
	if cmd == "keys" {
		doKeys(fs.Args())
		return
	}

	ctx := context.Background()
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

	switch cmd {
	case "search":
		doSearch(ctx, engine, fs.Args())
	case "fetch":
		doFetch(ctx, engine, fs.Args())
	case "serve":
		doServe(ctx, engine, fs.Args())
	default:
		usage()
		os.Exit(1)
	}
}

func doSearch(ctx context.Context, e saka.Searcher, args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	n := fs.Int("n", 10, "max results")
	format := fs.String("format", "table", "table|json|markdown")
	site := fs.String("site", "", "restrict to site")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fatal(fmt.Errorf("usage: saka search [flags] \"query\""))
	}

	res, err := e.Search(ctx, saka.Query{
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

func doFetch(ctx context.Context, e saka.Searcher, args []string) {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	format := fs.String("format", "text", "text|json|markdown")
	fs.Parse(args)
	if fs.NArg() != 1 {
		fatal(fmt.Errorf("usage: saka fetch [flags] <url>"))
	}

	page, err := e.Fetch(ctx, fs.Arg(0))
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

// doServe was added in the v0.2 pass ("CLI wiring (saka serve --mcp)")
// and gained --keys in the v1.1 "paid vs free" pass.
func doServe(ctx context.Context, e saka.Searcher, args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	mcp := fs.Bool("mcp", false, "serve over stdio as an MCP server")
	addr := fs.String("addr", ":8080", "HTTP listen address")
	keysPath := fs.String("keys", "", "path to api_keys.json (enables auth)")
	fs.Parse(args)

	if *mcp {
		if err := server.NewMCP(e).ServeStdio(ctx); err != nil {
			fatal(err)
		}
		return
	}

	opts := server.Options{}
	if *keysPath != "" {
		b, err := os.ReadFile(*keysPath)
		if err != nil {
			fatal(err)
		}
		var keys map[string]string // {"sk-...": "pro", ...}
		if err := json.Unmarshal(b, &keys); err != nil {
			fatal(err)
		}
		opts.Keys = server.StaticKeys(keys)
	}
	h := server.NewWithOptions(e, opts).Handler() // REST API, from v0.2 + v1.1 auth
	fmt.Printf("saka listening on %s\n", *addr)
	fatal(http.ListenAndServe(*addr, h))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "saka:", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `saka — free web search, no API keys

Usage:
  saka search "query" [-n 10] [--format table|json|markdown] [--site example.com]
  saka fetch <url> [--format text|json|markdown]
  saka serve [--addr :8080] [--mcp] [--config saka.json] [--keys api_keys.json]
  saka keys [--tier free|standard|pro] [--n 1] [--exp-days 365] [--priv path]

Flags:
  --config saka.json   configuration file
`)
}
