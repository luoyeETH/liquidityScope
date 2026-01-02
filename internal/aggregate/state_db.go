package aggregate

import (
	"context"

	"liquidityScope/internal/storage/postgres"
)

// DBStateStore stores state in the indexer_state table.
type DBStateStore struct {
	Store *postgres.Store
	Name  string
}

func (s *DBStateStore) Load(ctx context.Context) (uint64, bool, error) {
	if s == nil || s.Store == nil {
		return 0, false, nil
	}
	return s.Store.LoadState(ctx, s.Name)
}

func (s *DBStateStore) Save(ctx context.Context, ts uint64) error {
	if s == nil || s.Store == nil {
		return nil
	}
	return s.Store.SaveState(ctx, s.Name, ts)
}
