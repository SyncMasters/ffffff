package report

import (
	"github.com/johan-larp/agentsearch/internal/models"
	"time"
)

// DOCXReport preserves the existing generator API through the canonical pipeline.
type DOCXReport struct{}

func NewDOCXReport() *DOCXReport { return &DOCXReport{} }
func (*DOCXReport) Generate(target string, results []models.Result, duration time.Duration) (string, error) {
	return generateLegacy("docx", target, results, duration)
}
