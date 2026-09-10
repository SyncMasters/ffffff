package config

import (
	"flag"
	"testing"
	"time"
)

func TestServerConfiguration(t *testing.T) {
	cfg, e := ParseServerArgs(nil)
	if e != nil || cfg.Listen != "127.0.0.1:8080" || cfg.Engine.PasswordBackend != PasswordBackendAPI || cfg.MaxConcurrent != 8 || cfg.RequestTimeout != time.Minute {
		t.Fatal("unexpected defaults")
	}
	for _, args := range [][]string{{"-listen", "bad"}, {"-token-env", "invalid-name"}, {"-request-timeout", "0s"}, {"-write-timeout", "1s"}, {"-read-timeout", "0s"}, {"-max-concurrent", "0"}, {"-password-backend", "auto"}, {"-password-db", "private"}, {"-password-backend", "local"}, {"-password", "example-password"}, {"extra"}} {
		if _, e := ParseServerArgs(args); e == nil {
			t.Fatal("invalid server settings accepted")
		}
	}
	cfg, e = ParseServerArgs([]string{"-password-backend", "local", "-password-db", "operator-db", "-s", ""})
	if e != nil || cfg.Engine.PasswordDatabasePath != "operator-db" || cfg.Engine.SitesFile != "" {
		t.Fatal("explicit backend config lost")
	}
	if _, e = ParseServerArgs([]string{"-h"}); e != flag.ErrHelp {
		t.Fatal("help unsupported")
	}
}
