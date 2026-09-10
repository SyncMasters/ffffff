package storage

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/johan-larp/agentsearch/internal/models"
)

// CSVWriter пишет результаты в CSV с заголовком.
type CSVWriter struct {
	file *os.File
	w    *csv.Writer
	mu   sync.Mutex
}

// NewCSVWriter создает CSV-файл и записывает заголовок.
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

// Write преобразует Result в CSV-запись.
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

// Close завершает запись.
func (w *CSVWriter) Close() error {
	w.w.Flush()
	return w.file.Close()
}
