package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/johan-larp/agentsearch/internal/app"
	"github.com/johan-larp/agentsearch/internal/config"
)

func main() {
	// Use structured logging on stderr.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.ParseFlags()
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	slog.Info("agentsearch starting",
		"targets", len(cfg.Targets),
		"workers", cfg.Workers,
		"sites", cfg.SitesFile,
		"proxies", cfg.ProxiesFile,
		"output", cfg.OutputDir,
		"formats", cfg.OutputFormats,
	)

	// Apply a deadline to the entire run.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.TotalTimeout)
	defer cancel()

	application, err := app.New(cfg)
	if err != nil {
		slog.Error("initialization error", "error", err)
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		slog.Error("runtime error", "error", err)
		os.Exit(1)
	}

	slog.Info("agentsearch finished successfully")
}
