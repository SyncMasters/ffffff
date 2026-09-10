// Package app constructs dependencies and orchestrates searches and output.
package app

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/johan-larp/agentsearch/internal/config"
	"github.com/johan-larp/agentsearch/internal/network"
	"github.com/johan-larp/agentsearch/internal/ratelimit"
	"github.com/johan-larp/agentsearch/internal/security"
	"github.com/johan-larp/agentsearch/internal/sources"
	"github.com/johan-larp/agentsearch/internal/sources/hibp"
	"github.com/johan-larp/agentsearch/internal/sources/websites"
)

type App struct {
	cfg    *config.AppConfig
	runner *Runner
}

// New constructs only the providers selected by the CLI mode.
func New(cfg *config.AppConfig) (*App, error) {
	if cfg.Mode == config.ModeEmail {
		return newEmailApp(cfg)
	}
	if cfg.Mode != config.ModeWebsites {
		return nil, fmt.Errorf("unsupported search mode")
	}
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

// newEmailApp is a composition root, not an alternate search path. Execution
// still goes through Runner and the capability dispatcher.
func newEmailApp(cfg *config.AppConfig) (*App, error) {
	settings, err := config.LoadServices(cfg.ServicesFile)
	if err != nil {
		return nil, err
	}
	service, ok := settings.Services["hibp"]
	if !ok || !service.Enabled {
		return nil, fmt.Errorf("HIBP email lookup is disabled; set services.hibp.enabled to true in the services file")
	}
	keyEnv := service.APIKeyEnv
	if keyEnv == "" {
		keyEnv = config.DefaultHIBPKeyEnv
	}
	key := security.NewSecret(os.Getenv(keyEnv))
	if key.Reveal() == "" {
		return nil, fmt.Errorf("HIBP email lookup requires the %s environment variable", keyEnv)
	}
	client, err := hibp.NewClient(service.APIURL, key, network.NewServiceClient(cfg.RequestTimeout))
	if err != nil {
		return nil, err
	}
	source, err := hibp.NewSource(client)
	if err != nil {
		return nil, err
	}
	registry := sources.NewRegistry()
	if err := registry.Register(source); err != nil {
		return nil, err
	}
	return NewWithSources(cfg, registry), nil
}
