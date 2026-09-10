package ratelimit

import (
	"context"
	"net/url"
	"sync"
	"time"
)

// HostLimiter реализует задержку между запросами к одному хосту,
// чтобы минимизировать 429 Too Many Requests и баны.
type HostLimiter struct {
	delay       time.Duration
	mu          sync.Mutex
	lastRequest map[string]time.Time
}

// NewHostLimiter создает лимитер. delay=0 отключает ограничения.
func NewHostLimiter(delay time.Duration) *HostLimiter {
	return &HostLimiter{
		delay:       delay,
		lastRequest: make(map[string]time.Time),
	}
}

// Wait блокирует вызывающую горутину до тех пор, пока не пройдет
// достаточно времени с момента предыдущего запроса к этому хосту.
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
