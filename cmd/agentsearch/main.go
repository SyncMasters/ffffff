package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/johan-larp/agentsearch/internal/app"
	"github.com/johan-larp/agentsearch/internal/config"
	"github.com/johan-larp/agentsearch/internal/security"
)

func main() {
	// Use structured logging on stderr.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	os.Exit(run(context.Background(), os.Args[1:], security.ReadPasswordPrompt))
}

// run returns an exit code rather than exiting so secrets and terminal state
// are cleaned before main calls os.Exit. The prompt is injectable for CI.
func run(parent context.Context, args []string, prompt security.PasswordPrompt) (exitCode int) {
	cfg, err := config.ParseArgs(args)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		slog.Error("configuration error", "error", err)
		return 1
	}
	defer func() { cfg.Password.Destroy() }()
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, cfg.TotalTimeout)
	defer cancel()

	var application *app.App
	defer func() {
		if application != nil {
			if err := application.Close(); err != nil {
				slog.Error("application cleanup failed", "error", err)
				exitCode = 1
			}
		}
	}()
	if cfg.Mode == config.ModePassword {
		method := "sha1-k-anonymity"
		if cfg.PasswordBackend == config.PasswordBackendLocal {
			method = "sha1-offline"
			application, err = app.NewWithContext(ctx, cfg)
			if err != nil {
				slog.Error("initialization error", "error", err)
				return 1
			}
		}
		slog.Info("password lookup starting", "source", "Pwned Passwords", "method", method)
		if cfg.PasswordPrompt {
			if prompt == nil {
				slog.Error("password prompt is unavailable")
				return 1
			}
			cfg.Password, err = prompt(ctx)
			if err != nil {
				slog.Error("password input failed; use an interactive terminal and complete input before the timeout")
				return 1
			}
			if cfg.Password.Empty() {
				slog.Error("password input must not be empty")
				return 1
			}
		} else {
			slog.Warn("-password exposes input through shell history, process listings, terminal logs and auditing; prefer -password-prompt")
		}
	} else {
		slog.Info("agentsearch starting", "targets", len(cfg.Targets), "workers", cfg.Workers, "sites", cfg.SitesFile, "proxies", cfg.ProxiesFile, "output", cfg.OutputDir, "formats", cfg.OutputFormats)
	}
	if application == nil {
		application, err = app.New(cfg)
	}
	if err != nil {
		slog.Error("initialization error", "error", err)
		return 1
	}
	if err := application.Run(ctx); err != nil {
		slog.Error("runtime error", "error", err)
		return 1
	}
	slog.Info("agentsearch finished successfully")
	return 0
}
