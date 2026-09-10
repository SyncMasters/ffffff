package storage

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/johan-larp/agentsearch/internal/models"
)

// JSONWriter реализует потоковую запись JSON-массива.
// Файл открывается как [elem1, elem2, ...] без необходимости держать всё в памяти.
type JSONWriter struct {
	file  *os.File
	mu    sync.Mutex
	first bool
}

// NewJSONWriter создает JSON-файл и пишет открывающую скобку массива.
func NewJSONWriter(path string) (*JSONWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if _, err := f.WriteString("[\n"); err != nil {
		return nil, err
	}
	return &JSONWriter{file: f, first: true}, nil
}

// Write сериализует результат и дописывает в файл с запятой-разделителем.
func (w *JSONWriter) Write(res models.Result) error {
	res = res.Normalized()
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(res)
	if err != nil {
		return err
	}
	if !w.first {
		if _, err := w.file.WriteString(",\n"); err != nil {
			return err
		}
	}
	if _, err := w.file.Write(data); err != nil {
		return err
	}
	w.first = false
	return nil
}

// Close закрывает массив и файл.
func (w *JSONWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.file.WriteString("\n]\n"); err != nil {
		return err
	}
	return w.file.Close()
}
