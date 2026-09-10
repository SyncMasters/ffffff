package storage

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/johan-larp/agentsearch/internal/models"
)

// CSVWriter writes results with the legacy CSV header.
type CSVWriter struct {
	file *os.File
	w    *csv.Writer
	mu   sync.Mutex
}

// NewCSVWriter creates a CSV file and writes its header.
func NewCSVWriter(path string) (*CSVWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	writer := csv.NewWriter(f)
	header := []string{"site_name", "target", "url", "found", "confidence", "status", "duration_ms", "error", "final_url"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	writer.Flush()
	return &CSVWriter{file: f, w: writer}, nil
}

// Write projects a normalized result into a CSV record.
func (w *CSVWriter) Write(res models.Result) error {
	res = res.Normalized()
	w.mu.Lock()
	defer w.mu.Unlock()

	record := []string{
		res.SiteName,
		res.Target,
		res.URL,
		strconv.FormatBool(res.Found),
		strconv.Itoa(res.Confidence),
		string(res.Status),
		fmt.Sprintf("%d", res.Duration.Milliseconds()),
		res.Error,
		res.FinalURL,
	}
	if err := w.w.Write(record); err != nil {
		return err
	}
	w.w.Flush()
	return w.w.Error()
}

// Close flushes and closes the file.
func (w *CSVWriter) Close() error {
	w.w.Flush()
	return w.file.Close()
}
