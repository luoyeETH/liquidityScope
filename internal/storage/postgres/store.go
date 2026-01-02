package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"liquidityScope/internal/model"
)

// Store provides Postgres persistence for metrics.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("pg dsn is required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// UpsertPools inserts or updates pool metadata.
func (s *Store) UpsertPools(ctx context.Context, pools []model.Pool) error {
	if len(pools) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, pool := range pools {
		batch.Queue(`
			INSERT INTO pools (
				chain_id, pool_address, token0, token1, fee, tick_spacing, first_seen_block, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
			ON CONFLICT (chain_id, pool_address)
			DO UPDATE SET
				token0 = EXCLUDED.token0,
				token1 = EXCLUDED.token1,
				fee = EXCLUDED.fee,
				tick_spacing = EXCLUDED.tick_spacing,
				first_seen_block = LEAST(pools.first_seen_block, EXCLUDED.first_seen_block),
				updated_at = now()
		`,
			int64(pool.ChainID),
			pool.Address,
			pool.Token0,
			pool.Token1,
			pool.Fee,
			pool.TickSpacing,
			int64(pool.FirstSeenBlock),
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range pools {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// UpsertWindowMetrics inserts or updates window metrics.
func (s *Store) UpsertWindowMetrics(ctx context.Context, metrics []model.PoolWindowMetrics) error {
	if len(metrics) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, m := range metrics {
		batch.Queue(`
			INSERT INTO pool_window_metrics (
				chain_id, pool_address, window_size_seconds, window_start_ts, window_end_ts,
				swap_count, volume0, volume1, fee0, fee1, fee_usd, fee_rate0, fee_rate1,
				tvl0, tvl1, tvl_usd, apr, fee_method, tvl_method, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,now(),now())
			ON CONFLICT (chain_id, pool_address, window_size_seconds, window_start_ts)
			DO UPDATE SET
				window_end_ts = EXCLUDED.window_end_ts,
				swap_count = EXCLUDED.swap_count,
				volume0 = EXCLUDED.volume0,
				volume1 = EXCLUDED.volume1,
				fee0 = EXCLUDED.fee0,
				fee1 = EXCLUDED.fee1,
				fee_usd = EXCLUDED.fee_usd,
				fee_rate0 = EXCLUDED.fee_rate0,
				fee_rate1 = EXCLUDED.fee_rate1,
				tvl0 = EXCLUDED.tvl0,
				tvl1 = EXCLUDED.tvl1,
				tvl_usd = EXCLUDED.tvl_usd,
				apr = EXCLUDED.apr,
				fee_method = EXCLUDED.fee_method,
				tvl_method = EXCLUDED.tvl_method,
				updated_at = now()
		`,
			int64(m.ChainID),
			m.PoolAddress,
			m.WindowSizeSecs,
			m.WindowStart,
			m.WindowEnd,
			int64(m.SwapCount),
			m.Volume0,
			m.Volume1,
			m.Fee0,
			m.Fee1,
			m.FeeUSD,
			m.FeeRate0,
			m.FeeRate1,
			m.TVL0,
			m.TVL1,
			m.TVLUSD,
			m.APR,
			m.FeeMethod,
			m.TVLMethod,
		)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range metrics {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// LoadState returns last_processed_ts for a name.
func (s *Store) LoadState(ctx context.Context, name string) (uint64, bool, error) {
	if name == "" {
		return 0, false, fmt.Errorf("state name required")
	}
	var ts uint64
	row := s.pool.QueryRow(ctx, `SELECT last_processed_ts FROM indexer_state WHERE name=$1`, name)
	if err := row.Scan(&ts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return ts, true, nil
}

// SaveState upserts last_processed_ts for a name.
func (s *Store) SaveState(ctx context.Context, name string, ts uint64) error {
	if name == "" {
		return fmt.Errorf("state name required")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO indexer_state (name, last_processed_ts, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (name) DO UPDATE
		SET last_processed_ts = EXCLUDED.last_processed_ts, updated_at = now()
	`, name, ts)
	return err
}
