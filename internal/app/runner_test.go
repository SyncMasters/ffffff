package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/johan-larp/agentsearch/internal/config"
	"github.com/johan-larp/agentsearch/internal/models"
	"github.com/johan-larp/agentsearch/internal/sources"
)

type localTestSource struct{}

func (localTestSource) Name() string            { return "local-test" }
func (localTestSource) Type() models.SourceType { return models.SourceLocal }
func (localTestSource) SearchEmail(ctx context.Context, email string, emit sources.Emit) error {
	return emit(models.Result{Status: models.StatusFound, Metadata: map[string]string{"observation": "fixture"}})
}
func TestApplicationWithoutWebsiteInfrastructure(t *testing.T) {
	registry := sources.NewRegistry()
	if err := registry.Register(localTestSource{}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg := &config.AppConfig{Targets: []string{"alice@example.test"}, OutputDir: dir, OutputFormats: []string{"json", "csv", "txt"}}
	// No site database, network client, proxy, limiter or workers are required.
	a := NewWithSources(cfg, registry)
	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "alice@example.test.json"))
	if err != nil {
		t.Fatal(err)
	}
	var results []models.Result
	if err := json.Unmarshal(b, &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source != "local-test" || results[0].TargetType != models.TargetEmail || results[0].Metadata["observation"] != "fixture" {
		t.Fatal("generic path failed", results)
	}
}

func TestHIBPConfiguration(t *testing.T) {
	dir := t.TempDir()
	services := filepath.Join(dir, "services.yaml")
	t.Setenv(config.DefaultHIBPKeyEnv, "")
	cfg := &config.AppConfig{Mode: config.ModeEmail, ServicesFile: services, Targets: []string{"alice@example.test"}}
	if _, err := New(cfg); err == nil {
		t.Fatal("missing services file accepted")
	}
	if err := os.WriteFile(services, []byte("services:\n  hibp:\n    enabled: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(cfg); err == nil {
		t.Fatal("disabled service accepted")
	}
	if err := os.WriteFile(services, []byte("services:\n  hibp:\n    enabled: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(cfg); err == nil {
		t.Fatal("missing API key accepted")
	}
}
