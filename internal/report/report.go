package report

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/johan-larp/agentsearch/internal/models"
	"github.com/johan-larp/agentsearch/internal/security"
)

// Generator создаёт отчёт по результатам поиска.
type Generator interface {
	Generate(target string, results []models.Result, duration time.Duration) (string, error)
}

// Summary содержит агрегированную статистику по результатам.
type Summary struct {
	Target       string
	Total        int
	Found        int
	NotFound     int
	Blocked      int
	Errors       int
	Duration     time.Duration
	TopSites     []SiteStat
	ByTag        map[string]TagStat
	BlockedHosts []string
}

// SiteStat — статистика по одному сайту.
type SiteStat struct {
	Name       string
	Found      int
	Blocked    int
	AvgLatency time.Duration
}

// TagStat — статистика по тегу.
type TagStat struct {
	Tag         string
	Total       int
	Found       int
	Blocked     int
	SuccessRate float64
}

// BuildSummary агрегирует результаты в сводку.
func BuildSummary(target string, results []models.Result, duration time.Duration) Summary {
	target, results = normalizeResults(target, results)
	s := Summary{
		Target:       target,
		Total:        len(results),
		Duration:     duration,
		ByTag:        make(map[string]TagStat),
		BlockedHosts: make([]string, 0),
	}

	siteMap := make(map[string]*SiteStat)
	blockedSet := make(map[string]struct{})

	for _, r := range results {
		switch r.Status {
		case models.StatusFound:
			s.Found++
		case models.StatusNotFound:
			s.NotFound++
		case models.StatusBlocked:
			s.Blocked++
			blockedSet[r.SiteName] = struct{}{}
		case models.StatusError:
			s.Errors++
		}

		// Агрегация по сайтам
		st, ok := siteMap[r.SiteName]
		if !ok {
			st = &SiteStat{Name: r.SiteName}
			siteMap[r.SiteName] = st
		}
		if r.Status == models.StatusFound {
			st.Found++
		}
		if r.Status == models.StatusBlocked {
			st.Blocked++
		}
		st.AvgLatency += r.Duration
	}

	for _, st := range siteMap {
		if st.Found > 0 || st.Blocked > 0 {
			st.AvgLatency = st.AvgLatency / time.Duration(st.Found+st.Blocked+1)
			s.TopSites = append(s.TopSites, *st)
		}
	}
	sort.Slice(s.TopSites, func(i, j int) bool {
		return s.TopSites[i].Found > s.TopSites[j].Found
	})

	for host := range blockedSet {
		s.BlockedHosts = append(s.BlockedHosts, host)
	}
	sort.Strings(s.BlockedHosts)

	return s
}

// WriteAll генерирует все запрошенные форматы отчётов.
func WriteAll(dir string, formats []string, target string, results []models.Result, duration time.Duration) error {
	target, results = normalizeResults(target, results)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create report dir: %w", err)
	}

	summary := BuildSummary(target, results, duration)

	for _, f := range formats {
		var gen Generator
		switch f {
		case "cli":
			gen = NewCLIReport()
		case "html":
			gen = NewHTMLReport()
		case "docx":
			gen = NewDOCXReport()
		default:
			slog.Warn("unknown report format", "format", f)
			continue
		}

		path, err := gen.Generate(target, results, duration)
		if err != nil {
			slog.Error("generate report failed", "format", f, "error", err)
			continue
		}
		slog.Info("report generated", "format", f, "path", path)
	}

	// Также пишем summary JSON
	summaryPath := filepath.Join(dir, fmt.Sprintf("%s_summary.json", sanitizeFilename(target)))
	if err := writeSummaryJSON(summaryPath, summary); err != nil {
		slog.Error("write summary failed", "error", err)
	}

	return nil
}

func writeSummaryJSON(path string, summary Summary) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(struct {
		Target   string `json:"target"`
		Total    int    `json:"total"`
		Found    int    `json:"found"`
		NotFound int    `json:"not_found"`
		Blocked  int    `json:"blocked"`
		Errors   int    `json:"errors"`
		Duration string `json:"duration"`
	}{summary.Target, summary.Total, summary.Found, summary.NotFound, summary.Blocked, summary.Errors, summary.Duration.String()})
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_",
	)
	return replacer.Replace(name)
}

// normalizeResults protects direct generator callers as well as the app path.
// The legacy string argument is a display label, never sensitive input.
func normalizeResults(target string, results []models.Result) (string, []models.Result) {
	var secrets []string
	for _, r := range results {
		if r.TargetType.Sensitive() {
			secrets = append(secrets, target, r.Target)
			target = security.Redacted
		}
	}
	out := make([]models.Result, len(results))
	for i, r := range results {
		out[i] = r.Redacted(secrets...).Normalized()
	}
	return security.Redact(target, secrets...), out
}
