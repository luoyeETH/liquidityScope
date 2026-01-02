package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"

	"liquidityScope/internal/chain"
	"liquidityScope/internal/model"
	"liquidityScope/internal/storage"
)

// RunConfig holds runtime settings for the indexer.
type RunConfig struct {
	FromBlock         uint64
	ToBlock           uint64
	Addresses         []common.Address
	Topic0            []common.Hash
	BatchSize         uint64
	CheckpointPath    string
	CheckpointEnabled bool
	MaxRetries        int
	RetryBackoff      time.Duration
}

// Runner streams logs from the chain and writes them to storage.
type Runner struct {
	cfg        RunConfig
	chain      *chain.Client
	storage    storage.Storage
	logger     *zap.Logger
	seen       map[string]struct{}
	checkpoint *CheckpointStore
}

// NewRunner builds a Runner with its dependencies.
func NewRunner(cfg RunConfig, chainClient *chain.Client, storageSink storage.Storage, logger *zap.Logger) *Runner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Runner{
		cfg:        cfg,
		chain:      chainClient,
		storage:    storageSink,
		logger:     logger,
		seen:       make(map[string]struct{}),
		checkpoint: NewCheckpointStore(cfg.CheckpointPath, cfg.CheckpointEnabled),
	}
}

// Run executes the indexing loop.
func (r *Runner) Run(ctx context.Context) error {
	if r.chain == nil {
		return fmt.Errorf("chain client is nil")
	}
	if r.storage == nil {
		return fmt.Errorf("storage is nil")
	}
	if r.cfg.BatchSize == 0 {
		return fmt.Errorf("batch size must be greater than zero")
	}
	if len(r.cfg.Addresses) == 0 {
		return fmt.Errorf("at least one address is required")
	}

	chainID, err := r.chain.GetChainID(ctx)
	if err != nil {
		return fmt.Errorf("get chain id: %w", err)
	}
	if !chainID.IsUint64() {
		return fmt.Errorf("chain id does not fit in uint64: %s", chainID)
	}
	chainIDValue := chainID.Uint64()

	from := r.cfg.FromBlock
	to := r.cfg.ToBlock
	if to == 0 {
		latest, err := r.chain.LatestBlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("get latest block: %w", err)
		}
		to = latest
	}

	if r.checkpoint != nil {
		cp, ok, err := r.checkpoint.Load()
		if err != nil {
			return err
		}
		if ok && cp.LastProcessedBlock >= from {
			from = cp.LastProcessedBlock + 1
			r.logger.Info("resume from checkpoint", zap.Uint64("last_processed", cp.LastProcessedBlock), zap.Uint64("from", from))
		}
	}

	if from > to {
		r.logger.Info("nothing to sync", zap.Uint64("from", from), zap.Uint64("to", to))
		return nil
	}

	ranges, err := SplitRange(from, to, r.cfg.BatchSize)
	if err != nil {
		return err
	}

	for _, blockRange := range ranges {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		r.logger.Info("fetch logs", zap.Uint64("from", blockRange.From), zap.Uint64("to", blockRange.To))

		logs, err := r.filterLogsWithRetry(ctx, blockRange.From, blockRange.To)
		if err != nil {
			return fmt.Errorf("filter logs: %w", err)
		}

		ingestedAt := time.Now().UTC()
		records := make([]model.LogRecord, 0, len(logs))
		for _, log := range logs {
			if r.isDuplicate(log) {
				continue
			}

			ts, err := r.blockTimestampWithRetry(ctx, log.BlockNumber)
			if err != nil {
				return fmt.Errorf("block timestamp %d: %w", log.BlockNumber, err)
			}
			records = append(records, buildLogRecord(chainIDValue, log, ts, ingestedAt))
		}

		if err := r.storage.PutLogBatch(records); err != nil {
			return fmt.Errorf("store logs: %w", err)
		}

		if r.checkpoint != nil {
			if err := r.checkpoint.Save(blockRange.To); err != nil {
				return err
			}
		}

		r.logger.Info("batch complete", zap.Int("logs", len(records)), zap.Uint64("from", blockRange.From), zap.Uint64("to", blockRange.To))
	}

	return nil
}

func (r *Runner) filterLogsWithRetry(ctx context.Context, fromBlock, toBlock uint64) ([]types.Log, error) {
	var logs []types.Log
	err := withRetry(ctx, r.cfg.MaxRetries, r.cfg.RetryBackoff, func(ctx context.Context) error {
		var err error
		logs, err = r.chain.FilterLogs(ctx, fromBlock, toBlock, r.cfg.Addresses, r.cfg.Topic0)
		if err != nil {
			r.logger.Warn("filter logs failed", zap.Error(err), zap.Uint64("from", fromBlock), zap.Uint64("to", toBlock))
		}
		return err
	})
	return logs, err
}

func (r *Runner) blockTimestampWithRetry(ctx context.Context, blockNumber uint64) (uint64, error) {
	var ts uint64
	err := withRetry(ctx, r.cfg.MaxRetries, r.cfg.RetryBackoff, func(ctx context.Context) error {
		var err error
		ts, err = r.chain.BlockTimestamp(ctx, blockNumber)
		if err != nil {
			r.logger.Warn("block timestamp fetch failed", zap.Error(err), zap.Uint64("block_number", blockNumber))
		}
		return err
	})
	return ts, err
}

func (r *Runner) isDuplicate(log types.Log) bool {
	id := fmt.Sprintf("%d:%s:%d", log.BlockNumber, log.TxHash.Hex(), log.Index)
	if _, ok := r.seen[id]; ok {
		return true
	}
	r.seen[id] = struct{}{}
	return false
}
