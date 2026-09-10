package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitContext(t *testing.T) {
	limiter := NewHostLimiter(20 * time.Millisecond)
	limiter.Wait("https://example.test/first")
	start := time.Now()
	limiter.Wait("https://example.test/second")
	if time.Since(start) < 15*time.Millisecond {
		t.Fatal("rate limit lost")
	}
	limiter = NewHostLimiter(time.Hour)
	limiter.Wait("http://example.test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(limiter.WaitContext(ctx, "http://example.test"), context.Canceled) {
		t.Fatal("cancellation ignored")
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if !errors.Is(limiter.WaitContext(ctx, "http://example.test"), context.DeadlineExceeded) {
		t.Fatal("timer not cancellable")
	}
}
