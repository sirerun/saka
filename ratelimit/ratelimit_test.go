package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestLimiterAllowsBurstThenWaits(t *testing.T) {
	l := New(100)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := l.Wait(ctx); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLimiterRespectsCancel(t *testing.T) {
	l := New(0.5) // starts below 1 token, so Wait must sleep
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := l.Wait(ctx); err == nil {
		t.Fatal("expected context error")
	}
}

func TestNewDefaultsZeroRPS(t *testing.T) {
	l := New(0)
	if l.rate != 1 {
		t.Fatalf("rate = %v", l.rate)
	}
}
