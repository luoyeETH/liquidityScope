package aggregate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StateStore persists the last processed timestamp.
type StateStore interface {
	Load(ctx context.Context) (uint64, bool, error)
	Save(ctx context.Context, ts uint64) error
}

// FileStateStore stores state in a local JSON file.
type FileStateStore struct {
	Path string
}

type stateRecord struct {
	LastProcessed uint64 `json:"last_processed_ts"`
	UpdatedAt     string `json:"updated_at"`
}

func (s *FileStateStore) Load(ctx context.Context) (uint64, bool, error) {
	if s == nil || s.Path == "" {
		return 0, false, nil
	}
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("read state: %w", err)
	}

	var rec stateRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return 0, false, fmt.Errorf("parse state: %w", err)
	}
	return rec.LastProcessed, true, nil
}

func (s *FileStateStore) Save(ctx context.Context, ts uint64) error {
	if s == nil || s.Path == "" {
		return nil
	}
	dir := filepath.Dir(s.Path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create state dir: %w", err)
		}
	}

	rec := stateRecord{
		LastProcessed: ts,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write state tmp: %w", err)
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}
