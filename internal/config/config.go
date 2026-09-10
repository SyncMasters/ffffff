package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/johan-larp/agentsearch/internal/models"
)

type SearchMode string

const (
	ModeWebsites SearchMode = ""
	ModeEmail    SearchMode = "email"
)

// AppConfig holds CLI options and runtime settings.
type AppConfig struct {
	Mode                SearchMode
	ServicesFile        string
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
func ParseFlags() (*AppConfig, error) { return ParseArgs(os.Args[1:]) }

// ParseArgs preserves legacy flags while separating email breach lookups from
// website searches. A fresh FlagSet makes parsing repeatable in tests.
func ParseArgs(args []string) (*AppConfig, error) {
	flags := flag.NewFlagSet("agentsearch", flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: agentsearch -u TARGET | -f FILE | -email ADDRESS [options]")
		fmt.Fprintln(flags.Output(), "Website flags never query HIBP. Email lookups require enabled service settings and an API key.")
		flags.PrintDefaults()
	}
	var (
		email    = flags.String("email", "", "Email breach lookup through HIBP (not a website search)")
		services = flags.String("services", "configs/services.yaml", "Service configuration for -email")
		u        = flags.String("u", "", "Target username or email")
		f        = flags.String("f", "", "File with targets (one per line)")
		s        = flags.String("s", "configs/sites.yaml", "Sites database YAML/JSON")
		p        = flags.String("p", "", "Proxies file (http://ip:port or socks5://ip:port)")
		o        = flags.String("o", "output", "Output directory")
		of       = flags.String("of", "json,csv,txt", "Output formats: json,csv,txt (comma-separated)")
		rf       = flags.String("rf", "cli,html", "Report formats: cli,html,docx (comma-separated)")
		w        = flags.Int("w", 50, "Number of concurrent workers")
		rt       = flags.Duration("rt", 15*time.Second, "HTTP request timeout")
		tt       = flags.Duration("tt", 10*time.Minute, "Total search timeout per batch")
		mc       = flags.Int("mc", 500, "Max idle connections in pool")
		mch      = flags.Int("mch", 100, "Max idle connections per host")
		rl       = flags.Duration("rl", 500*time.Millisecond, "Rate limit delay between requests to same host")
		ua       = flags.String("ua", "", "External User-Agent list file")
		d        = flags.Bool("d", false, "Enable deep search (dorking mode stub)")
		utls     = flags.Bool("utls", false, "Enable uTLS JA3 fingerprint spoofing (anti-WAF)")
		retries  = flags.Int("retries", 2, "Max retries on 429/5xx errors")
	)
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	selectedEmail := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "email" {
			selectedEmail = true
		}
	})
	if selectedEmail {
		allowed := map[string]bool{"email": true, "services": true, "rt": true, "tt": true, "o": true, "of": true, "rf": true}
		var unsupported string
		flags.Visit(func(f *flag.Flag) {
			if !allowed[f.Name] {
				unsupported = f.Name
			}
		})
		if unsupported != "" {
			return nil, fmt.Errorf("-%s is a website option and cannot be combined with -email", unsupported)
		}
		if flags.NArg() != 0 {
			return nil, fmt.Errorf("-email accepts one address; unexpected positional arguments")
		}
		if *rt <= 0 || *tt <= 0 {
			return nil, fmt.Errorf("email lookup timeouts must be positive")
		}
	} else {
		usedServices := false
		flags.Visit(func(f *flag.Flag) {
			if f.Name == "services" {
				usedServices = true
			}
		})
		if usedServices {
			return nil, fmt.Errorf("-services is only used with -email")
		}
	}

	if !selectedEmail && *u == "" && *f == "" {
		return nil, fmt.Errorf("usage: agentsearch -u TARGET OR -f FILE OR -email ADDRESS")
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

	if selectedEmail {
		target, err := models.NewEmailTarget(*email)
		if err != nil {
			return nil, err
		}
		cfg.Mode = ModeEmail
		cfg.ServicesFile = *services
		cfg.Targets = []string{target.Value()}
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
