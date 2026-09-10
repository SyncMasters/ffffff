package app

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/johan-larp/agentsearch/internal/config"
	"github.com/johan-larp/agentsearch/internal/diagnostics"
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
func (r *Runner) Search(ctx context.Context, target models.Target, emit sources.Emit) (err error) {
	start := time.Now()
	var summary diagnostics.Summary
	completed := false
	invalidTarget := !target.Valid()
	var consumerErr error
	diagnostics.Started(ctx, "search", target.Type(), "")
	defer func() {
		outcome := summary.Finish(ctx, err)
		if invalidTarget {
			outcome = diagnostics.Failure("invalid_request")
		}
		if errors.Is(err, sources.ErrUnsupportedTarget) {
			outcome = diagnostics.Failure("invalid_request")
		}
		if emit == nil {
			outcome = diagnostics.Failure("internal_error")
		}
		if consumerErr != nil {
			outcome = diagnostics.Failure(diagnostics.ErrorCategory(consumerErr, "internal_error"))
		}
		if !completed {
			outcome = diagnostics.Failure("internal_error")
		}
		diagnostics.Completed(ctx, "search", target.Type(), "", start, outcome)
	}()
	if emit == nil {
		err = r.manager.Search(ctx, target, nil)
	} else {
		err = r.manager.Search(ctx, target, func(result models.Result) error {
			summary.Observe(result)
			consumerErr = emit(result)
			return consumerErr
		})
	}
	completed = true
	return err
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
		if a.cfg.Mode == config.ModeDomain {
			target, err = models.NewDomainTarget(value)
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
			// Preserve the legacy website exit policy; service modes report failures.
			if a.cfg.Mode == config.ModeEmail || a.cfg.Mode == config.ModeDomain {
				return err
			}
		}
	}
	return nil
}

func (a *App) searchTarget(ctx context.Context, target models.Target) error {
	start := time.Now()
	store, err := storage.NewManager(a.cfg.OutputDir, a.cfg.OutputFormats, target.String())
	if err != nil {
		slog.Error("output initialization failed", "operation", "output_setup", "target_type", diagnostics.Target(target.Type()), "outcome", "error", "error_category", "internal_error")
		return err
	}
	defer store.Close()
	var results []models.Result
	searchErr := a.runner.Search(ctx, target, func(res models.Result) error {
		store.Write(res)
		results = append(results, res)
		return nil
	})
	elapsed := time.Since(start)
	if len(a.cfg.ReportFormats) > 0 {
		if err := report.WriteAll(a.cfg.OutputDir, a.cfg.ReportFormats, target.String(), results, elapsed); err != nil {
			slog.Error("report generation failed", "error_category", "internal_error")
		}
	}
	return searchErr
}
