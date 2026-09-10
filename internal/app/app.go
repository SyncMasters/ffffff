// Package app constructs dependencies and orchestrates searches and output.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/johan-larp/agentsearch/internal/config"
	"github.com/johan-larp/agentsearch/internal/network"
	"github.com/johan-larp/agentsearch/internal/passworddb"
	"github.com/johan-larp/agentsearch/internal/ratelimit"
	"github.com/johan-larp/agentsearch/internal/security"
	"github.com/johan-larp/agentsearch/internal/sources"
	"github.com/johan-larp/agentsearch/internal/sources/hibp"
	"github.com/johan-larp/agentsearch/internal/sources/websites"
)

type App struct {
	cfg    *config.AppConfig
	runner *Runner
	close  func() error
}

// New constructs only the providers selected by the CLI mode.
func New(cfg *config.AppConfig) (*App, error) { return NewWithContext(context.Background(), cfg) }

// NewWithContext permits cancellable local snapshot preparation before prompting.
// Existing API/website constructors and dispatch behavior remain unchanged.
func NewWithContext(ctx context.Context, cfg *config.AppConfig) (*App, error) {
	if cfg.Mode == config.ModePassword {
		var application *App
		var err error
		switch cfg.PasswordBackend {
		case config.PasswordBackendLocal:
			application, err = newLocalPasswordApp(ctx, cfg)
		case "", config.PasswordBackendAPI:
			if cfg.PasswordDatabasePath != "" {
				err = fmt.Errorf("database path requires local password backend")
			} else {
				application, err = newPasswordApp(cfg)
			}
		default:
			err = fmt.Errorf("password backend must be api or local")
		}
		if err != nil {
			cfg.Password.Destroy()
		}
		return application, err
	}
	if cfg.Mode == config.ModeDomain {
		return newDomainApp(cfg)
	}
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

// Password composition never resolves email credentials or builds website workers.
func newPasswordApp(cfg *config.AppConfig) (*App, error) {
	rangeURL := ""
	if cfg.ServicesFile != "" {
		settings, err := config.LoadServices(cfg.ServicesFile)
		if err != nil {
			return nil, err
		}
		rangeURL = settings.Services["hibp"].PasswordsAPIURL
	}
	client, err := hibp.NewPasswordClient(rangeURL, network.NewServiceClient(cfg.RequestTimeout))
	if err != nil {
		return nil, err
	}
	source, err := hibp.NewPasswordSource(client)
	if err != nil {
		return nil, err
	}
	registry := sources.NewRegistry()
	if err := registry.Register(source); err != nil {
		return nil, err
	}
	return NewWithSources(cfg, registry), nil
}

// Close releases app-owned local resources; other source lifecycles are unchanged.
func (a *App) Close() error {
	if a.close != nil {
		return a.close()
	}
	return nil
}

func newLocalPasswordApp(ctx context.Context, cfg *config.AppConfig) (*App, error) {
	path := cfg.PasswordDatabasePath
	if path == "" {
		var err error
		path, err = config.LocalPasswordPath(cfg)
		if err != nil {
			return nil, err
		}
	}
	snapshot, err := passworddb.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	source, err := hibp.NewLocalPasswordSource(snapshot)
	if err != nil {
		snapshot.Close()
		return nil, err
	}
	registry := sources.NewRegistry()
	if err = registry.Register(source); err != nil {
		snapshot.Close()
		return nil, err
	}
	application := NewWithSources(cfg, registry)
	application.close = snapshot.Close
	if snapshot.Info().Recovered {
		slog.Warn("offline database activation recovered from redundant record", "dataset_id", snapshot.Info().DatasetID)
	}
	return application, nil
}
