package saka

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sirerun/saka/types"
)

func TestHomeDir(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		t.Setenv("HOME", "/tmp/somewhere")
		if got := homeDir(); got != "/tmp/somewhere" {
			t.Errorf("homeDir() = %q, want /tmp/somewhere", got)
		}
	})
	t.Run("unset", func(t *testing.T) {
		t.Setenv("HOME", "")
		if got := homeDir(); got != "" {
			t.Errorf("homeDir() = %q, want empty on lookup failure", got)
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := LoadConfig(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(path); err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid.json")
		if err := os.WriteFile(path, []byte(`{"providers":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(path); err == nil {
			t.Fatal("expected validate error")
		}
	})

	t.Run("valid", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ok.json")
		body := `{"providers":[{"name":"duckduckgo","rps":1,"retries":1}],"fetch":{"rps":1}}`
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg.Providers) != 1 || cfg.Providers[0].Name != "duckduckgo" {
			t.Errorf("unexpected config: %+v", cfg)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("invalid config", func(t *testing.T) {
		if _, err := New(Config{}); err == nil {
			t.Fatal("expected error for empty config")
		}
	})

	t.Run("all providers no cache", func(t *testing.T) {
		cfg := Config{
			Providers: []ProviderConfig{
				{Name: "duckduckgo", RPS: 1, Retries: 1},
				{Name: "searxng", URL: "http://localhost:8888", RPS: 1, Retries: 1},
				{Name: "startpage", RPS: 1, Retries: 1},
			},
			Fetch: FetchConfig{RPS: 1},
		}
		e, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if e == nil {
			t.Fatal("expected non-nil engine")
		}
	})

	t.Run("disk cache with tilde and zero ttl", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		cfg := Config{
			Providers: []ProviderConfig{{Name: "duckduckgo", RPS: 1}},
			Fetch: FetchConfig{
				RPS:       1,
				DiskCache: &DiskCacheConfig{Dir: "~/sakacache"},
			},
		}
		e, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if e == nil {
			t.Fatal("expected non-nil engine")
		}
		want := filepath.Join(home, "sakacache")
		if _, statErr := os.Stat(want); statErr != nil {
			t.Errorf("expected disk cache dir at %s: %v", want, statErr)
		}
	})

	t.Run("disk cache mkdir failure", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := Config{
			Providers: []ProviderConfig{{Name: "duckduckgo", RPS: 1}},
			Fetch: FetchConfig{
				RPS:       1,
				DiskCache: &DiskCacheConfig{Dir: filepath.Join(blocker, "sub")},
			},
		}
		if _, err := New(cfg); err == nil {
			t.Fatal("expected error when disk cache dir can't be created")
		}
	})
}

func TestEngineMethods(t *testing.T) {
	e, err := New(Config{
		Providers: []ProviderConfig{{Name: "duckduckgo", RPS: 1}},
		Fetch:     FetchConfig{RPS: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("search", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		if _, err := e.Search(ctx, Query{Text: "q"}); err == nil {
			t.Fatal("expected context deadline error")
		}
	})

	t.Run("fetch", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		if _, err := e.Fetch(ctx, "https://example.com"); err == nil {
			t.Fatal("expected context deadline error")
		}
	})

	t.Run("fetch stream", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		_, _, errCh := e.FetchStream(ctx, "https://example.com")
		if err := <-errCh; err == nil {
			t.Fatal("expected context deadline error")
		}
	})
}

type fakeSearchStreamProvider struct{}

func (fakeSearchStreamProvider) Name() string { return "fake-searchstream" }

func (fakeSearchStreamProvider) Search(_ context.Context, _ types.Query) ([]types.Result, error) {
	return []types.Result{
		{Title: "one", URL: "https://one.example", Position: 1},
		{Title: "two", URL: "https://two.example", Position: 2},
		{Title: "three", URL: "https://three.example", Position: 3},
	}, nil
}

func TestEngineSearchStream(t *testing.T) {
	if err := types.Register("fake-searchstream", func(types.ProviderConfig) (types.Provider, error) {
		return fakeSearchStreamProvider{}, nil
	}); err != nil {
		t.Fatal(err)
	}

	e, err := New(Config{
		Providers: []ProviderConfig{{Name: "fake-searchstream", RPS: 1}},
		Fetch:     FetchConfig{RPS: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	q := Query{Text: "q"}

	want, err := e.Search(ctx, q)
	if err != nil {
		t.Fatal(err)
	}

	itemCh, doneCh, errCh := e.SearchStream(ctx, q)

	var got []Result
	for r := range itemCh {
		got = append(got, r)
	}

	select {
	case err := <-errCh:
		t.Fatalf("unexpected error on error channel: %v", err)
	default:
	}

	done := <-doneCh
	if done == nil {
		t.Fatal("expected non-nil *Results on done channel")
	}
	if !reflect.DeepEqual(got, want.Results) {
		t.Errorf("streamed results = %+v, want %+v", got, want.Results)
	}
	if done.Provider != want.Provider {
		t.Errorf("done.Provider = %q, want %q", done.Provider, want.Provider)
	}
	if done.Query != want.Query {
		t.Errorf("done.Query = %q, want %q", done.Query, want.Query)
	}
	if done.TookMs < 0 {
		t.Errorf("done.TookMs = %d, want >= 0", done.TookMs)
	}
}
