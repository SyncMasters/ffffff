// Package websites adapts the existing site database engine into one provider.
package websites

import (
	"context"
	"fmt"
	"github.com/johan-larp/agentsearch/internal/detector"
	"github.com/johan-larp/agentsearch/internal/models"
	"github.com/johan-larp/agentsearch/internal/network"
	"github.com/johan-larp/agentsearch/internal/ratelimit"
	"github.com/johan-larp/agentsearch/internal/sources"
	"github.com/johan-larp/agentsearch/internal/worker"
	"log/slog"
	"net/http"
)

type Source struct {
	sites    []models.SiteConfig
	workers  int
	client   *http.Client
	ua       *network.UARotator
	detector *detector.Engine
	limiter  *ratelimit.HostLimiter
}

// New reuses the existing HTTP client, detector, limiter and worker pool.
// Site configurations are read-only for the source's lifetime.
func New(sites []models.SiteConfig, workers int, client *http.Client, ua *network.UARotator, limiter *ratelimit.HostLimiter) (*Source, error) {
	if workers < 1 {
		return nil, fmt.Errorf("website workers must be positive")
	}
	if client == nil || ua == nil || limiter == nil {
		return nil, fmt.Errorf("missing website dependency")
	}
	return &Source{sites: append([]models.SiteConfig(nil), sites...), workers: workers, client: client, ua: ua, detector: detector.New(), limiter: limiter}, nil
}
func (*Source) Name() string            { return "websites" }
func (*Source) Type() models.SourceType { return models.SourceWebsite }
func (s *Source) SearchUsername(ctx context.Context, username string, emit sources.Emit) error {
	return s.search(ctx, models.TargetUsername, username, emit)
}

// SearchEmail retains the legacy behavior: substitute the email in the site
// templates, without claiming that every configured website supports emails.
func (s *Source) SearchEmail(ctx context.Context, email string, emit sources.Emit) error {
	return s.search(ctx, models.TargetEmail, email, emit)
}
func (s *Source) search(ctx context.Context, kind models.TargetType, value string, emit sources.Emit) error {
	if emit == nil {
		return fmt.Errorf("nil result consumer")
	}
	target, err := models.NewTarget(kind, value)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	pool := worker.NewPool(s.workers, &siteProcessor{source: s, target: target.Value(), kind: kind})
	pool.Start(ctx)
	submitted := make(chan struct{})
	go func() {
		defer close(submitted)
		defer pool.Close()
		count := 0
		for _, site := range s.sites {
			if site.URL == "" || site.Disabled {
				continue
			}
			if err := pool.SubmitContext(ctx, worker.Job{Site: site, Target: value}); err != nil {
				break
			}
			count++
		}
		slog.Info("jobs submitted", "target", target, "count", count)
	}()
	var consumerErr error
	for res := range pool.Results() {
		if consumerErr == nil {
			consumerErr = emit(res)
			if consumerErr != nil {
				cancel()
			}
		}
	}
	<-submitted
	if consumerErr != nil {
		return consumerErr
	}
	return ctx.Err()
}

var _ sources.Source = (*Source)(nil)
var _ sources.UsernameSearcher = (*Source)(nil)
var _ sources.EmailSearcher = (*Source)(nil)
