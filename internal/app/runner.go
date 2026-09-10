package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/johan-larp/agentsearch/internal/config"
	"github.com/johan-larp/agentsearch/internal/models"
	"github.com/johan-larp/agentsearch/internal/report"
	"github.com/johan-larp/agentsearch/internal/sources"
	"github.com/johan-larp/agentsearch/internal/storage"
)

// Runner executes typed searches without CLI, signals or output policy.
type Runner struct{ manager *sources.Manager }

func NewRunner(registry *sources.Registry) *Runner {
	return &Runner{manager: sources.NewManager(registry)}
}
func (r *Runner) Search(ctx context.Context, target models.Target, emit sources.Emit) error {
	return r.manager.Search(ctx, target, emit)
}

// Run searches all CLI targets and handles graceful shutdown.
func (a *App) Run(ctx context.Context) error {
	// Cancel pending work on SIGINT or SIGTERM.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	if a.cfg.Mode == config.ModePassword {
		defer a.cfg.Password.Destroy()
		target, err := models.NewSensitiveTarget(models.TargetPassword, a.cfg.Password)
		if err != nil {
			return err
		}
		return a.searchTarget(ctx, target)
	}

	for _, value := range a.cfg.Targets {
		target, err := models.LegacyTarget(value)
		if a.cfg.Mode == config.ModeEmail {
			target, err = models.NewEmailTarget(value)
		}
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			slog.Warn("shutdown signal received, exiting")
			return ctx.Err()
		default:
		}
		if err := a.searchTarget(ctx, target); err != nil {
			slog.Error("search target failed", "target", target, "error", err)
			// Preserve the legacy website exit policy; new email mode reports failures.
			if a.cfg.Mode == config.ModeEmail {
				return err
			}
		}
	}
	return nil
}

func (a *App) searchTarget(ctx context.Context, target models.Target) error {
	slog.Info("starting search", "target", target, "workers", a.cfg.Workers)
	start := time.Now()
	store, err := storage.NewManager(a.cfg.OutputDir, a.cfg.OutputFormats, target.String())
	if err != nil {
		return err
	}
	defer store.Close()
	var results []models.Result
	searchErr := a.runner.Search(ctx, target, func(res models.Result) error {
		store.Write(res)
		a.logResult(res)
		results = append(results, res)
		return nil
	})
	elapsed := time.Since(start)
	slog.Info("search completed", "target", target, "elapsed", elapsed.Round(time.Second), "results", len(results))
	if len(a.cfg.ReportFormats) > 0 {
		if err := report.WriteAll(a.cfg.OutputDir, a.cfg.ReportFormats, target.String(), results, elapsed); err != nil {
			slog.Error("report generation failed", "error", err)
		}
	}
	return searchErr
}
func (a *App) logResult(res models.Result) {
	if res.TargetType == models.TargetPassword {
		label := "ERROR"
		if res.Status == models.StatusFound {
			label = "PWNED"
		} else if res.Status == models.StatusNotFound {
			label = "NOT PWNED"
		}
		slog.Info(label, "source", res.SiteName, "method", res.Metadata["method"], "occurrences", res.Metadata["occurrences"], "error", res.Error)
		return
	}
	switch res.Status {
	case models.StatusFound:
		slog.Info("match found", "source", res.SiteName, "url", res.URL, "confidence", res.Confidence)
	case models.StatusBlocked:
		slog.Warn("source blocked", "source", res.SiteName, "url", res.URL)
	case models.StatusError:
		slog.Debug("lookup error", "source", res.SiteName, "error", res.Error)
	default:
		slog.Debug("no match found", "source", res.SiteName, "url", res.URL)
	}
}
