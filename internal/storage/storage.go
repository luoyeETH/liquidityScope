package storage

import "liquidityScope/internal/model"

// Storage defines a sink for log records.
type Storage interface {
	PutLogBatch(logs []model.LogRecord) error
}
