package report

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/johan-larp/agentsearch/internal/models"
)

func TestGenericReports(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("output", 0755); err != nil {
		t.Fatal(err)
	}
	results := []models.Result{{Source: "local-test", SourceType: models.SourceLocal, Target: "never-print-this", TargetType: models.TargetPassword, Status: models.StatusFound, Metadata: map[string]string{"safe-detail": "observation", "password": "never-print-this"}, Evidence: []models.Evidence{{Kind: "observation", Value: "<script>alert(1)</script>"}}}}
	for _, tt := range []struct {
		name string
		gen  Generator
	}{{"cli", NewCLIReport()}, {"html", NewHTMLReport()}, {"docx", NewDOCXReport()}} {
		t.Run(tt.name, func(t *testing.T) {
			path, err := tt.gen.Generate("never-print-this", results, time.Second)
			if err != nil {
				if tt.name == "docx" && (strings.Contains(strings.ToLower(err.Error()), "license") || strings.Contains(strings.ToLower(err.Error()), "unlicensed")) {
					t.Skipf("existing UniOffice license requirement: %v", err)
				}
				t.Fatal(err)
			}
			var data []byte
			if tt.name == "docx" {
				z, err := zip.OpenReader(path)
				if err != nil {
					t.Fatal(err)
				}
				defer z.Close()
				for _, f := range z.File {
					if f.Name == "word/document.xml" {
						r, err := f.Open()
						if err != nil {
							t.Fatal(err)
						}
						data, err = io.ReadAll(r)
						_ = r.Close()
						if err != nil {
							t.Fatal(err)
						}
					}
				}
			} else {
				data, err = os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
			}
			if strings.Contains(string(data), "never-print-this") || !strings.Contains(string(data), "local-test") {
				t.Fatal("unsafe or missing generic result")
			}
			if tt.name == "html" && (!strings.Contains(string(data), "safe-detail") || strings.Contains(string(data), "<script>alert(1)</script>")) {
				t.Fatal("missing or unescaped generic details")
			}
		})
	}
}
func TestSummaryJSONEscaping(t *testing.T) {
	path := t.TempDir() + "/summary.json"
	target := "alice\"\\\n"
	if err := writeSummaryJSON(path, BuildSummary(target, []models.Result{{Source: "local", Status: models.StatusFound}}, time.Second)); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatal(err)
	}
	if result["target"] != target || result["found"] != float64(1) {
		t.Fatal("summary changed", result)
	}
}
