package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// AppConfig holds CLI options and runtime settings.
type AppConfig struct {
	Targets             []string
	SitesFile           string
	ProxiesFile         string
	OutputDir           string
	OutputFormats       []string
	ReportFormats       []string
	Workers             int
	RequestTimeout      time.Duration
	TotalTimeout        time.Duration
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	RateLimitPerHost    time.Duration
	UserAgentFile       string
	DeepSearch          bool
	UseUTLS             bool
	MaxRetries          int
}

// ParseFlags parses command-line options.
func ParseFlags() (*AppConfig, error) {
	var (
		u       = flag.String("u", "", "Target username or email")
		f       = flag.String("f", "", "File with targets (one per line)")
		s       = flag.String("s", "configs/sites.yaml", "Sites database YAML/JSON")
		p       = flag.String("p", "", "Proxies file (http://ip:port or socks5://ip:port)")
		o       = flag.String("o", "output", "Output directory")
		of      = flag.String("of", "json,csv,txt", "Output formats: json,csv,txt (comma-separated)")
		rf      = flag.String("rf", "cli,html", "Report formats: cli,html,docx (comma-separated)")
		w       = flag.Int("w", 50, "Number of concurrent workers")
		rt      = flag.Duration("rt", 15*time.Second, "HTTP request timeout")
		tt      = flag.Duration("tt", 10*time.Minute, "Total search timeout per batch")
		mc      = flag.Int("mc", 500, "Max idle connections in pool")
		mch     = flag.Int("mch", 100, "Max idle connections per host")
		rl      = flag.Duration("rl", 500*time.Millisecond, "Rate limit delay between requests to same host")
		ua      = flag.String("ua", "", "External User-Agent list file")
		d       = flag.Bool("d", false, "Enable deep search (dorking mode stub)")
		utls    = flag.Bool("utls", false, "Enable uTLS JA3 fingerprint spoofing (anti-WAF)")
		retries = flag.Int("retries", 2, "Max retries on 429/5xx errors")
	)
	flag.Parse()

	if *u == "" && *f == "" {
		return nil, fmt.Errorf("usage: ./agentsearch -u target OR -f targets.txt")
	}

	cfg := &AppConfig{
		SitesFile:           *s,
		ProxiesFile:         *p,
		OutputDir:           *o,
		Workers:             *w,
		RequestTimeout:      *rt,
		TotalTimeout:        *tt,
		MaxIdleConns:        *mc,
		MaxIdleConnsPerHost: *mch,
		RateLimitPerHost:    *rl,
		UserAgentFile:       *ua,
		DeepSearch:          *d,
		UseUTLS:             *utls,
		MaxRetries:          *retries,
	}

	if *u != "" {
		cfg.Targets = append(cfg.Targets, *u)
	}
	if *f != "" {
		data, err := os.ReadFile(*f)
		if err != nil {
			return nil, fmt.Errorf("read targets file: %w", err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				cfg.Targets = append(cfg.Targets, line)
			}
		}
	}

	for _, format := range strings.Split(*of, ",") {
		format = strings.TrimSpace(strings.ToLower(format))
		if format != "" {
			cfg.OutputFormats = append(cfg.OutputFormats, format)
		}
	}
	for _, format := range strings.Split(*rf, ",") {
		format = strings.TrimSpace(strings.ToLower(format))
		if format != "" {
			cfg.ReportFormats = append(cfg.ReportFormats, format)
		}
	}

	return cfg, nil
}
