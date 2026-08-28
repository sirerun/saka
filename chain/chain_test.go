package chain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirerun/saka/types"
)

type fakeProvider struct {
	name  string
	err   error
	calls *int
}

func (f fakeProvider) Name() string { return f.name }
func (f fakeProvider) Search(_ context.Context, _ types.Query) ([]types.Result, error) {
	*f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return []types.Result{{Title: "hit", URL: "https://x", Source: f.name, Position: 1}}, nil
}

func TestChainFallsBackOnRateLimit(t *testing.T) {
	calls1, calls2 := 0, 0
	p1 := fakeProvider{name: "one", err: &types.RateLimitError{Provider: "one"}, calls: &calls1}
	p2 := fakeProvider{name: "two", calls: &calls2}

	c := New(
		[]types.ProviderConfig{{Name: "one"}, {Name: "two"}},
		[]types.Provider{p1, p2},
	)
	res, err := c.Search(context.Background(), types.Query{Text: "q"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Provider != "two" {
		t.Errorf("expected fallback to two, got %s", res.Provider)
	}
	if calls2 != 1 {
		t.Errorf("provider two not called exactly once: %d", calls2)
	}
}

func TestChainBreakerOpens(t *testing.T) {
	calls := 0
	p := fakeProvider{name: "flaky", err: errors.New("boom"), calls: &calls}
	c := New(
		[]types.ProviderConfig{{Name: "flaky"}, {Name: "ok"}},
		[]types.Provider{p, fakeProvider{name: "ok", calls: new(int)}},
	)
	// burn the breaker: 3 failures
	for i := 0; i < breakerThreshold; i++ {
		c.Search(context.Background(), types.Query{Text: "q"})
	}
	before := calls
	c.Search(context.Background(), types.Query{Text: "q"})
	if calls != before {
		t.Errorf("breaker open but flaky provider was called")
	}
	// after cooldown it should be tried again
	for _, e := range c.entries {
		e.openUntil = time.Now().Add(-time.Second)
	}
	c.Search(context.Background(), types.Query{Text: "q"})
	if calls == before {
		t.Error("breaker never reopened")
	}
}

func TestChainSuccessShortCircuits(t *testing.T) {
	calls1 := 0
	c := New(
		[]types.ProviderConfig{{Name: "a"}, {Name: "b"}},
		[]types.Provider{
			fakeProvider{name: "a", calls: &calls1},
			fakeProvider{name: "b", calls: new(int)},
		},
	)
	res, err := c.Search(context.Background(), types.Query{Text: "q"})
	if err != nil || res.Provider != "a" {
		t.Fatalf("first provider should win: %v %v", res, err)
	}
}
