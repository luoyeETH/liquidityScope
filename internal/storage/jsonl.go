package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"liquidityScope/internal/model"
)

// JsonlStorage writes log records to a JSONL file.
type JsonlStorage struct {
	path string
	mu   sync.Mutex
}

func NewJsonlStorage(path string) *JsonlStorage {
	return &JsonlStorage{path: path}
}

// PutLogBatch appends a batch of log records as JSON lines.
func (s *JsonlStorage) PutLogBatch(logs []model.LogRecord) error {
	if len(logs) == 0 {
		return nil
	}

	dir := filepath.Dir(s.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open output file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, record := range logs {
		line, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("marshal log record: %w", err)
		}
		if _, err := writer.Write(line); err != nil {
			return fmt.Errorf("write log record: %w", err)
		}
		if err := writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
}
