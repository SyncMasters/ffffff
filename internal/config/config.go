package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/johan-larp/agentsearch/internal/models"
	"github.com/johan-larp/agentsearch/internal/security"
)

const (
	PasswordBackendAPI   = "api"
	PasswordBackendLocal = "local"
)

type SearchMode string

const (
	ModeWebsites SearchMode = ""
	ModeEmail    SearchMode = "email"
	ModeDomain   SearchMode = "domain"
	ModePassword SearchMode = "password"
)

// AppConfig holds CLI options and runtime settings.
type AppConfig struct {
	Mode                 SearchMode
	Password             security.Secret `json:"-" yaml:"-"`
	PasswordPrompt       bool
	PasswordBackend      string
	PasswordDatabasePath string
	ServicesFile         string
	Targets              []string
	SitesFile            string
	ProxiesFile          string
	OutputDir            string
	OutputFormats        []string
	ReportFormats        []string
	Workers              int
	RequestTimeout       time.Duration
	TotalTimeout         time.Duration
	MaxIdleConns         int
	MaxIdleConnsPerHost  int
	RateLimitPerHost     time.Duration
	UserAgentFile        string
	DeepSearch           bool
	UseUTLS              bool
	MaxRetries           int
}

// ParseFlags parses command-line options.
func ParseFlags() (*AppConfig, error) { return ParseArgs(os.Args[1:]) }

// ParseArgs preserves legacy flags while separating email breach lookups from
// website searches. A fresh FlagSet makes parsing repeatable in tests.
func ParseArgs(args []string) (cfg *AppConfig, parseErr error) {
	password := &passwordFlag{}
	defer func() {
		if cfg == nil {
			password.value.Destroy()
		}
	}()
	flags := flag.NewFlagSet("agentsearch", flag.ContinueOnError)
	sensitiveArgs := false
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if name == "password" || name == "password-prompt" {
			sensitiveArgs = true
		}
	}
	// flag's diagnostics may echo arbitrary arguments. Keep errors out of logs;
	// explicit help is restored below and never prints supplied values.
	flags.SetOutput(io.Discard)
	flags.Var(password, "password", "Single Pwned Passwords lookup; exposes the argument to shell history/process listings; prefer -password-prompt")
	passwordPrompt := flags.Bool("password-prompt", false, "Read one password from an interactive terminal without echo (recommended)")
	backend, database := PasswordBackendAPI, ""
	usedBackend, usedDatabase := false, false
	flags.Func("password-backend", "Password backend: api (default) or local; never automatically falls back", func(value string) error {
		if usedBackend {
			return fmt.Errorf("password backend must be supplied only once")
		}
		usedBackend = true
		backend = value
		return nil
	})
	flags.Func("password-db", "Managed offline database root; requires explicit local backend", func(value string) error {
		if usedDatabase {
			return fmt.Errorf("password database must be supplied only once")
		}
		usedDatabase = true
		database = value
		return nil
	})
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: agentsearch -u TARGET | -f FILE | -email ADDRESS | -domain HOSTNAME | -password VALUE | -password-prompt [options]")
		fmt.Fprintln(flags.Output(), "Website flags never query HIBP. Email lookups require enabled service settings and an API key.")
		fmt.Fprintln(flags.Output(), "Password API lookups require no API key and transmit only five SHA-1 characters. Explicit local mode makes no requests.")
		flags.PrintDefaults()
	}
	var (
		domain   = flags.String("domain", "", "Domain intelligence through configured external sources (not a website search)")
		email    = flags.String("email", "", "Email breach lookup through HIBP (not a website search)")
		services = flags.String("services", "configs/services.yaml", "Service configuration for -email, -domain or explicit password settings")
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
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(os.Stderr)
			flags.Usage()
			return nil, err
		}
		if sensitiveArgs {
			return nil, fmt.Errorf("invalid password lookup arguments; use -h for options")
		}
		return nil, err
	}
	selectedEmail, selectedDomain := false, false
	selectedPrompt, usedServices := false, false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "domain" {
			selectedDomain = true
		}
		if f.Name == "email" {
			selectedEmail = true
		}
		if f.Name == "password-prompt" {
			selectedPrompt = true
		}
		if f.Name == "services" {
			usedServices = true
		}
	})

	selectedPassword := password.seen || selectedPrompt
	if !selectedPassword && (usedBackend || usedDatabase) {
		return nil, fmt.Errorf("password backend options require password mode")
	}
	if selectedPassword && backend != PasswordBackendAPI && backend != PasswordBackendLocal {
		return nil, fmt.Errorf("password backend must be api or local")
	}
	if selectedPassword && usedDatabase && (backend != PasswordBackendLocal || database == "") {
		return nil, fmt.Errorf("password database requires explicit local mode and a nonempty path")
	}

	if selectedPassword {
		if password.seen && selectedPrompt || selectedPrompt && !*passwordPrompt {
			return nil, fmt.Errorf("choose exactly one of -password or -password-prompt")
		}
		if password.seen && password.value.Empty() {
			return nil, fmt.Errorf("password input must not be empty")
		}
		allowed := map[string]bool{"password": true, "password-prompt": true, "password-backend": true, "password-db": true, "services": true, "rt": true, "tt": true, "o": true, "of": true, "rf": true}
		invalid := false
		flags.Visit(func(f *flag.Flag) {
			if !allowed[f.Name] {
				invalid = true
			}
		})
		if invalid || flags.NArg() != 0 {
			return nil, fmt.Errorf("password mode accepts one input and only backend, database, services, timeout, output and report options")
		}
		if *rt <= 0 || *tt <= 0 {
			return nil, fmt.Errorf("password lookup timeouts must be positive")
		}
	} else if selectedDomain {
		allowed := map[string]bool{"domain": true, "services": true, "rt": true, "tt": true, "o": true, "of": true, "rf": true}
		invalid := false
		flags.Visit(func(f *flag.Flag) {
			if !allowed[f.Name] {
				invalid = true
			}
		})
		if invalid || flags.NArg() != 0 {
			return nil, fmt.Errorf("domain mode accepts one hostname and only services, timeout, output and report options")
		}
		if *rt <= 0 || *tt <= 0 {
			return nil, fmt.Errorf("domain lookup timeouts must be positive")
		}
	} else if selectedEmail {
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

		if usedServices {
			return nil, fmt.Errorf("-services is only used with -email, -domain or password mode")
		}
	}

	if !selectedEmail && !selectedDomain && !selectedPassword && *u == "" && *f == "" {
		return nil, fmt.Errorf("usage: agentsearch -u TARGET OR -f FILE OR -email ADDRESS OR -domain HOSTNAME OR -password VALUE OR -password-prompt")
	}

	cfg = &AppConfig{
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

	if selectedPassword {
		cfg.Mode = ModePassword
		cfg.Password = password.value
		cfg.PasswordPrompt = *passwordPrompt
		cfg.PasswordBackend = backend
		cfg.PasswordDatabasePath = database
		if usedServices {
			cfg.ServicesFile = *services
		}
	}
	if selectedDomain {
		target, err := models.NewDomainTarget(*domain)
		if err != nil {
			return nil, err
		}
		cfg.Mode = ModeDomain
		cfg.ServicesFile = *services
		cfg.Targets = []string{target.Value()}
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

	if cfg.Mode == ModePassword && cfg.PasswordBackend == PasswordBackendLocal {
		path, err := LocalPasswordPath(cfg)
		if err != nil {
			return nil, err
		}
		cfg.PasswordDatabasePath = path
	}
	if cfg.Mode == ModePassword {
		for _, format := range cfg.OutputFormats {
			if format != "json" && format != "csv" && format != "txt" {
				return nil, fmt.Errorf("invalid password output format; choose json,csv,txt")
			}
		}
		for _, format := range cfg.ReportFormats {
			if format != "cli" && format != "html" && format != "docx" {
				return nil, fmt.Errorf("invalid password report format; choose cli,html,docx")
			}
		}
	}
	return cfg, nil
}

// passwordFlag never retains an immutable copy or exposes its value in help.
type passwordFlag struct {
	value security.Secret
	seen  bool
}

func (*passwordFlag) String() string { return security.Redacted }
func (p *passwordFlag) Set(value string) error {
	if p.seen {
		return fmt.Errorf("password must be supplied only once")
	}
	p.seen = true
	p.value = security.NewSecret(value)
	return nil
}
