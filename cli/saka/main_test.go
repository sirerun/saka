package main

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	saka "github.com/sirerun/saka"
)

// timedWriter records the wall-clock time of each Write call, so tests can
// assert that output arrived incrementally rather than in one final burst.
type timedWriter struct {
	mu    sync.Mutex
	buf   strings.Builder
	times []time.Time
}

func (w *timedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.times = append(w.times, time.Now())
	return w.buf.Write(p)
}

func (w *timedWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// fakeStreamSearcher's SearchStream delivers items one at a time with an
// artificial delay between each send, simulating results arriving over
// time. Its Search method records whether it was called at all, so tests
// can prove --stream never falls back to the batched call.
type fakeStreamSearcher struct {
	searchCalls int32
	items       []saka.Result
	itemDelay   time.Duration
	final       *saka.Results
}

func (f *fakeStreamSearcher) Search(_ context.Context, q saka.Query) (*saka.Results, error) {
	atomic.AddInt32(&f.searchCalls, 1)
	return f.final, nil
}

func (f *fakeStreamSearcher) Fetch(_ context.Context, u string) (*saka.Page, error) {
	return &saka.Page{URL: u}, nil
}

func (f *fakeStreamSearcher) FetchStream(_ context.Context, _ string) (<-chan saka.Chunk, <-chan *saka.Page, <-chan error) {
	return nil, nil, nil
}

func (f *fakeStreamSearcher) SearchStream(_ context.Context, _ saka.Query) (<-chan saka.Result, <-chan *saka.Results, <-chan error) {
	itemCh := make(chan saka.Result)
	doneCh := make(chan *saka.Results, 1)
	errCh := make(chan error, 1)
	go func() {
		defer close(itemCh)
		for _, r := range f.items {
			time.Sleep(f.itemDelay)
			itemCh <- r
		}
		doneCh <- f.final
	}()
	return itemCh, doneCh, errCh
}

func TestRunSearchStreamPrintsIncrementally(t *testing.T) {
	items := []saka.Result{
		{Title: "One", URL: "https://one.example", Position: 1},
		{Title: "Two", URL: "https://two.example", Position: 2},
		{Title: "Three", URL: "https://three.example", Position: 3},
	}
	const delay = 40 * time.Millisecond
	f := &fakeStreamSearcher{
		items:     items,
		itemDelay: delay,
		final:     &saka.Results{Query: "q", Results: items, Provider: "fake"},
	}

	var w timedWriter
	newEngine := func(string) saka.Searcher { return f }
	if err := runSearch(newEngine, []string{"--stream", "q"}, &w); err != nil {
		t.Fatalf("runSearch: %v", err)
	}

	if atomic.LoadInt32(&f.searchCalls) != 0 {
		t.Fatalf("--stream called the batched Search method; want it to use SearchStream only")
	}

	if len(w.times) != len(items) {
		t.Fatalf("got %d writes, want %d (one per result)", len(w.times), len(items))
	}

	for i, want := range items {
		if !strings.Contains(w.buf.String(), want.Title) {
			t.Fatalf("output missing %q: %s", want.Title, w.buf.String())
		}
		_ = i
	}

	// Prove the writes were spread out over time (arrived as items streamed
	// in) rather than all happening back-to-back after a single blocking
	// wait for the full batch.
	gap := w.times[len(w.times)-1].Sub(w.times[0])
	minGap := delay * time.Duration(len(items)-1) / 2
	if gap < minGap {
		t.Fatalf("writes arrived too close together (gap %v, want >= %v); --stream appears to be buffering instead of draining incrementally", gap, minGap)
	}
}

func TestRunSearchDefaultPathUnchanged(t *testing.T) {
	final := &saka.Results{
		Query:    "q",
		Provider: "fake",
		Results: []saka.Result{
			{Title: "One", URL: "https://one.example", Snippet: "snip", Position: 1},
		},
	}
	f := &fakeStreamSearcher{final: final}

	var w timedWriter
	newEngine := func(string) saka.Searcher { return f }
	if err := runSearch(newEngine, []string{"q"}, &w); err != nil {
		t.Fatalf("runSearch: %v", err)
	}

	if atomic.LoadInt32(&f.searchCalls) != 1 {
		t.Fatalf("default (non-stream) path called Search %d times, want 1", f.searchCalls)
	}
	if !strings.Contains(w.String(), "One") || !strings.Contains(w.String(), "https://one.example") {
		t.Fatalf("output missing expected result: %s", w.String())
	}
}
