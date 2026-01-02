package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoint tracks the last processed block.
type Checkpoint struct {
	LastProcessedBlock uint64 `json:"last_processed_block"`
	UpdatedAt          string `json:"updated_at"`
}

// CheckpointStore persists checkpoints to disk.
type CheckpointStore struct {
	path    string
	enabled bool
}

func NewCheckpointStore(path string, enabled bool) *CheckpointStore {
	return &CheckpointStore{path: path, enabled: enabled}
}

func (c *CheckpointStore) Load() (Checkpoint, bool, error) {
	if !c.enabled {
		return Checkpoint{}, false, nil
	}

	stat, err := os.Stat(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Checkpoint{}, false, nil
		}
		return Checkpoint{}, false, fmt.Errorf("stat checkpoint: %w", err)
	}
	if stat.IsDir() {
		return Checkpoint{}, false, fmt.Errorf("checkpoint path is a directory")
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		return Checkpoint{}, false, fmt.Errorf("read checkpoint: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return Checkpoint{}, false, fmt.Errorf("parse checkpoint: %w", err)
	}

	return cp, true, nil
}

func (c *CheckpointStore) Save(lastProcessed uint64) error {
	if !c.enabled {
		return nil
	}

	dir := filepath.Dir(c.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create checkpoint dir: %w", err)
		}
	}

	cp := Checkpoint{
		LastProcessedBlock: lastProcessed,
		UpdatedAt:          time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}

	tmpPath := c.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write checkpoint tmp: %w", err)
	}
	if err := os.Rename(tmpPath, c.path); err != nil {
		return fmt.Errorf("rename checkpoint: %w", err)
	}

	return nil
}
