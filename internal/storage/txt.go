package storage

import (
	"fmt"
	"os"
	"sync"

	"github.com/johan-larp/agentsearch/internal/models"
)

// TXTWriter пишет человекочитаемый отчет.
type TXTWriter struct {
	file *os.File
	mu   sync.Mutex
}

// NewTXTWriter создает текстовый отчет.
func NewTXTWriter(path string) (*TXTWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &TXTWriter{file: f}, nil
}

// Write форматирует результат в строку.
func (w *TXTWriter) Write(res models.Result) error {
	res = res.Normalized()
	w.mu.Lock()
	defer w.mu.Unlock()

	line := fmt.Sprintf("[%s] %s | Found: %v | Confidence: %d%% | Status: %s | URL: %s\n",
		res.SiteName, res.Target, res.Found, res.Confidence, res.Status, res.URL)
	if res.FinalURL != "" && res.FinalURL != res.URL {
		line += fmt.Sprintf("  -> Final URL: %s\n", res.FinalURL)
	}
	_, err := w.file.WriteString(line)
	return err
}

// Close закрывает файл.
func (w *TXTWriter) Close() error {
	return w.file.Close()
}
