package ratelimit

import (
	"context"
	"net/url"
	"sync"
	"time"
)

// HostLimiter spaces requests to each host.
// Callers can cancel a pending delay using WaitContext.
type HostLimiter struct {
	delay       time.Duration
	mu          sync.Mutex
	lastRequest map[string]time.Time
}

// NewHostLimiter creates a limiter; zero delay disables waiting.
func NewHostLimiter(delay time.Duration) *HostLimiter {
	return &HostLimiter{
		delay:       delay,
		lastRequest: make(map[string]time.Time),
	}
}

// Wait blocks until the minimum interval for this host has elapsed.
func (rl *HostLimiter) Wait(rawURL string) { _ = rl.WaitContext(context.Background(), rawURL) }

// WaitContext is the cancellation-aware adapter used by website searches.
func (rl *HostLimiter) WaitContext(ctx context.Context, rawURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if rl.delay <= 0 {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	host := u.Host
	if host == "" {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if last, ok := rl.lastRequest[host]; ok {
		elapsed := time.Since(last)
		if elapsed < rl.delay {
			timer := time.NewTimer(rl.delay - elapsed)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	rl.lastRequest[host] = time.Now()
	return nil
}
