// Package app constructs dependencies and orchestrates searches and output.
package app

import (
	"fmt"
	"github.com/johan-larp/agentsearch/internal/config"
	"github.com/johan-larp/agentsearch/internal/network"
	"github.com/johan-larp/agentsearch/internal/ratelimit"
	"github.com/johan-larp/agentsearch/internal/sources"
	"github.com/johan-larp/agentsearch/internal/sources/websites"
	"log/slog"
)

type App struct {
	cfg    *config.AppConfig
	runner *Runner
}

// New constructs the legacy website-enabled CLI application.
func New(cfg *config.AppConfig) (*App, error) {
	ua, err := network.NewUARotator(cfg.UserAgentFile)
	if err != nil {
		return nil, fmt.Errorf("ua rotator: %w", err)
	}

	pr, err := network.NewProxyRotator(cfg.ProxiesFile)
	if err != nil {
		return nil, fmt.Errorf("proxy rotator: %w", err)
	}

	client := network.NewOptimizedClient(network.ClientConfig{
		RequestTimeout:      cfg.RequestTimeout,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		UseUTLS:             cfg.UseUTLS,
		MaxRetries:          cfg.MaxRetries,
	}, ua, pr)

	sites, err := config.LoadSites(cfg.SitesFile)
	if err != nil {
		return nil, fmt.Errorf("load sites: %w", err)
	}
	slog.Info("sites database loaded", "count", len(sites), "file", cfg.SitesFile)

	website, err := websites.New(sites, cfg.Workers, client, ua, ratelimit.NewHostLimiter(cfg.RateLimitPerHost))
	if err != nil {
		return nil, err
	}
	registry := sources.NewRegistry()
	if err := registry.Register(website); err != nil {
		return nil, err
	}
	return NewWithSources(cfg, registry), nil
}

// NewWithSources permits non-HTTP providers without constructing website infrastructure.
func NewWithSources(cfg *config.AppConfig, registry *sources.Registry) *App {
	return &App{cfg: cfg, runner: NewRunner(registry)}
}
