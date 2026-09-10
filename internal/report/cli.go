package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/johan-larp/agentsearch/internal/models"
)

// CLIReport renders a terminal summary and a plain-text report.
type CLIReport struct{}

func NewCLIReport() *CLIReport {
	return &CLIReport{}
}

func (c *CLIReport) Generate(target string, results []models.Result, duration time.Duration) (string, error) {
	target, results = normalizeResults(target, results)
	summary := BuildSummary(target, results, duration)

	// Print the terminal report.
	printHeader(target, duration)
	printStats(summary)
	writeDetails(os.Stdout, results)
	printFoundTable(results)
	printBlockedTable(results)
	printFooter(summary)

	// Save a plain-text copy.
	path := filepath.Join("output", fmt.Sprintf("%s_report.txt", sanitizeFilename(target)))
	if err := os.MkdirAll("output", 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Write the report header.
	fmt.Fprintf(f, "AGENTSEARCH REPORT\n")
	fmt.Fprintf(f, "==================\n\n")
	fmt.Fprintf(f, "Target: %s\n", target)
	fmt.Fprintf(f, "Duration: %s\n", duration.Round(time.Second))
	fmt.Fprintf(f, "Total checks: %d\n", summary.Total)
	fmt.Fprintf(f, "Found: %d | Not Found: %d | Blocked: %d | Errors: %d\n\n",
		summary.Found, summary.NotFound, summary.Blocked, summary.Errors)

	writeDetails(f, results)
	fmt.Fprintf(f, "FOUND RESULTS:\n")
	fmt.Fprintf(f, "%-30s %-10s %-10s %s\n", "SOURCE", "CONF", "STATUS", "URL")
	fmt.Fprintln(f, strings.Repeat("-", 120))
	for _, r := range results {
		if r.Status == models.StatusFound {
			fmt.Fprintf(f, "%-30s %-10d %-10s %s\n", r.SiteName, r.Confidence, r.Status, r.URL)
		}
	}

	fmt.Fprintf(f, "\nBLOCKED SOURCES:\n")
	for _, r := range results {
		if r.Status == models.StatusBlocked {
			fmt.Fprintf(f, "- %s (%s)\n", r.SiteName, r.URL)
		}
	}

	return path, nil
}

func printHeader(target string, duration time.Duration) {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	cyan.Printf("║           AGENTSEARCH OSINT REPORT                           ║\n")
	cyan.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("Target:    %s\n", color.YellowString(target))
	fmt.Printf("Duration:  %s\n", duration.Round(time.Second))
	fmt.Println()
}

func printStats(s Summary) {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	white := color.New(color.FgWhite)

	white.Printf("Statistics:\n")
	white.Printf("  Total checks:  %d\n", s.Total)
	green.Printf("  Found:         %d (%.1f%%)\n", s.Found, percent(s.Found, s.Total))
	white.Printf("  Not Found:     %d (%.1f%%)\n", s.NotFound, percent(s.NotFound, s.Total))
	yellow.Printf("  Blocked:       %d (%.1f%%)\n", s.Blocked, percent(s.Blocked, s.Total))
	red.Printf("  Errors:        %d (%.1f%%)\n", s.Errors, percent(s.Errors, s.Total))
	fmt.Println()
}

func printFoundTable(results []models.Result) {
	var found []models.Result
	for _, r := range results {
		if r.Status == models.StatusFound {
			found = append(found, r)
		}
	}
	if len(found) == 0 {
		color.Red("No successful matches to display.\n")
		return
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Printf("✓ Found %d result(s):\n\n", len(found))

	// Print table headings.
	fmt.Printf("%-30s %-12s %-12s %s\n", "SOURCE", "CONFIDENCE", "LATENCY", "URL")
	fmt.Println(strings.Repeat("─", 120))

	for _, r := range found {
		conf := fmt.Sprintf("%d%%", r.Confidence)
		if r.Confidence >= 70 {
			conf = color.GreenString(conf)
		} else if r.Confidence >= 40 {
			conf = color.YellowString(conf)
		} else {
			conf = color.WhiteString(conf)
		}
		fmt.Printf("%-30s %-12s %-12s %s\n",
			color.CyanString(r.SiteName),
			conf,
			r.Duration.Round(time.Millisecond).String(),
			r.URL,
		)
	}
	fmt.Println()
}

func printBlockedTable(results []models.Result) {
	var blocked []models.Result
	for _, r := range results {
		if r.Status == models.StatusBlocked {
			blocked = append(blocked, r)
		}
	}
	if len(blocked) == 0 {
		return
	}

	yellow := color.New(color.FgYellow, color.Bold)
	yellow.Printf("⚠ %d blocked source(s):\n", len(blocked))
	for _, r := range blocked {
		fmt.Printf("  • %s (%s)\n", r.SiteName, r.URL)
	}
	fmt.Println()
}

func printFooter(s Summary) {
	if s.Found > 0 {
		color.Green("✓ Search completed. Detailed reports saved to output/ directory.\n\n")
	} else if s.Errors > 0 {
		color.Yellow("Search finished with errors. Review the error results.\n\n")
	} else {
		color.Yellow("No matches returned by the selected sources.\n\n")
	}
}

func percent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

// writeDetails also represents successful no-match results and their metadata.
func writeDetails(w io.Writer, results []models.Result) {
	for _, result := range results {
		if len(result.Metadata) == 0 && len(result.Evidence) == 0 {
			continue
		}
		fmt.Fprintf(w, "%s | Source: %s\n", result.OutcomeLabel(), result.SiteName)
		for _, detail := range result.Details() {
			fmt.Fprintf(w, "  %s: %s\n", detail.Kind, detail.Value)
		}
	}
}
