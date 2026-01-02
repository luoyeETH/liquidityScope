package aggregate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	"liquidityScope/internal/chain"
	"liquidityScope/internal/model"
	"liquidityScope/internal/storage/postgres"
)

const (
	feeMethodApprox = "approx_from_feeTier"
	tvlMethodBlock  = "balance_of_block"
	tvlMethodLatest = "balance_of_latest"
	tvlMethodNone   = "unavailable"
)

// Config controls aggregation behavior.
type Config struct {
	WindowSeconds uint64
	BatchSize     int
	RecomputeFrom uint64
	StateStore    StateStore
}

// Aggregator aggregates typed events into pool window metrics.
type Aggregator struct {
	cfg          Config
	store        *postgres.Store
	chainClient  *chain.Client
	logger       *zap.Logger
	decimals     *TokenDecimalsCache
	accumulators map[string]*Accumulator
	poolSeen     map[string]model.Pool
}

func NewAggregator(cfg Config, store *postgres.Store, chainClient *chain.Client, logger *zap.Logger) *Aggregator {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Aggregator{
		cfg:          cfg,
		store:        store,
		chainClient:  chainClient,
		logger:       logger,
		decimals:     NewTokenDecimalsCache(),
		accumulators: make(map[string]*Accumulator),
		poolSeen:     make(map[string]model.Pool),
	}
}

// Run executes aggregation over a typed events JSONL file.
func (a *Aggregator) Run(ctx context.Context, inputPath string) error {
	if a.store == nil {
		return fmt.Errorf("store is nil")
	}
	if a.chainClient == nil {
		return fmt.Errorf("chain client is nil")
	}
	if a.cfg.WindowSeconds == 0 {
		return fmt.Errorf("window seconds must be > 0")
	}
	if a.cfg.BatchSize <= 0 {
		a.cfg.BatchSize = 1000
	}

	startTs, err := a.loadStartTimestamp(ctx)
	if err != nil {
		return err
	}

	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	batch := make([]model.PoolWindowMetrics, 0, a.cfg.BatchSize)
	pools := make([]model.Pool, 0, 256)
	maxTs := startTs
	var total, decoded, skipped, failed int

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		total++

		var record model.TypedEventRecord
		if err := json.Unmarshal(line, &record); err != nil {
			failed++
			a.logger.Warn("decode typed event", zap.Error(err))
			continue
		}

		if record.Timestamp <= startTs {
			skipped++
			continue
		}

		windowStart := windowStart(record.Timestamp, a.cfg.WindowSeconds)
		windowEnd := windowStart + a.cfg.WindowSeconds

		accKey := poolKey(record.Address)
		acc := a.accumulators[accKey]
		if acc == nil {
			acc = NewAccumulator(record, windowStart, windowEnd)
			a.accumulators[accKey] = acc
		} else if acc.WindowStart != windowStart {
			metrics, pool, err := a.flushAccumulator(ctx, acc)
			if err != nil {
				return err
			}
			if metrics != nil {
				batch = append(batch, *metrics)
				decoded++
			}
			if pool != nil {
				pools = append(pools, *pool)
			}
			acc = NewAccumulator(record, windowStart, windowEnd)
			a.accumulators[accKey] = acc
		}

		if err := acc.AddEvent(record); err != nil {
			failed++
			a.logger.Warn("aggregate event", zap.Error(err), zap.String("pool", record.Address), zap.String("event", record.EventName))
			continue
		}

		if record.Timestamp > maxTs {
			maxTs = record.Timestamp
		}

		if len(batch) >= a.cfg.BatchSize {
			if err := a.flushBatches(ctx, batch, pools); err != nil {
				return err
			}
			batch = batch[:0]
			pools = pools[:0]

			if err := a.saveState(ctx); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan input: %w", err)
	}

	for _, acc := range a.accumulators {
		metrics, pool, err := a.flushAccumulator(ctx, acc)
		if err != nil {
			return err
		}
		if metrics != nil {
			batch = append(batch, *metrics)
			decoded++
		}
		if pool != nil {
			pools = append(pools, *pool)
		}
	}
	a.accumulators = make(map[string]*Accumulator)

	if len(batch) > 0 || len(pools) > 0 {
		if err := a.flushBatches(ctx, batch, pools); err != nil {
			return err
		}
	}

	a.cfg.RecomputeFrom = maxTs
	if err := a.saveState(ctx); err != nil {
		return err
	}

	a.logger.Info("aggregate complete",
		zap.Int("total", total),
		zap.Int("decoded", decoded),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed),
	)

	return nil
}

func (a *Aggregator) loadStartTimestamp(ctx context.Context) (uint64, error) {
	if a.cfg.RecomputeFrom > 0 {
		return a.cfg.RecomputeFrom - 1, nil
	}
	if a.cfg.StateStore == nil {
		return 0, nil
	}
	last, ok, err := a.cfg.StateStore.Load(ctx)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	return last, nil
}

func (a *Aggregator) saveState(ctx context.Context) error {
	if a.cfg.StateStore == nil {
		return nil
	}

	if len(a.accumulators) == 0 {
		return a.cfg.StateStore.Save(ctx, a.cfg.RecomputeFrom)
	}

	safeTs := minOpenWindowStart(a.accumulators)
	if safeTs > 0 {
		safeTs = safeTs - 1
	}
	if safeTs == 0 {
		safeTs = a.cfg.RecomputeFrom
	}
	return a.cfg.StateStore.Save(ctx, safeTs)
}

func (a *Aggregator) flushBatches(ctx context.Context, batch []model.PoolWindowMetrics, pools []model.Pool) error {
	if len(pools) > 0 {
		if err := a.store.UpsertPools(ctx, pools); err != nil {
			return err
		}
	}
	if len(batch) > 0 {
		if err := a.store.UpsertWindowMetrics(ctx, batch); err != nil {
			return err
		}
	}
	return nil
}

func (a *Aggregator) flushAccumulator(ctx context.Context, acc *Accumulator) (*model.PoolWindowMetrics, *model.Pool, error) {
	if acc == nil {
		return nil, nil, nil
	}

	poolMeta := acc.PoolMeta
	if poolMeta.Token0 == "" || poolMeta.Token1 == "" {
		a.logger.Warn("missing pool meta", zap.String("pool", acc.PoolAddress))
		return nil, nil, nil
	}

	poolRecord := a.registerPool(acc)

	decimals0, err := a.getTokenDecimals(ctx, poolMeta.Token0)
	if err != nil {
		a.logger.Warn("token0 decimals", zap.String("token", poolMeta.Token0), zap.Error(err))
	}
	decimals1, err := a.getTokenDecimals(ctx, poolMeta.Token1)
	if err != nil {
		a.logger.Warn("token1 decimals", zap.String("token", poolMeta.Token1), zap.Error(err))
	}

	volume0 := formatTokenAmount(acc.Volume0, decimals0)
	volume1 := formatTokenAmount(acc.Volume1, decimals1)
	fee0 := formatTokenAmount(acc.Fee0, decimals0)
	fee1 := formatTokenAmount(acc.Fee1, decimals1)

	var tvl0Str, tvl1Str *string
	var tvl0Int, tvl1Int *big.Int
	var tvlMethod string
	if acc.LastBlock > 0 {
		balance0, balance1, method, err := a.fetchTVL(ctx, poolMeta.Token0, poolMeta.Token1, acc.PoolAddress, acc.LastBlock)
		if err != nil {
			a.logger.Warn("tvl fetch failed", zap.String("pool", acc.PoolAddress), zap.Error(err))
			tvlMethod = tvlMethodNone
		} else {
			tvl0Int = balance0
			tvl1Int = balance1
			if balance0 != nil {
				val := formatTokenAmount(balance0, decimals0)
				tvl0Str = &val
			}
			if balance1 != nil {
				val := formatTokenAmount(balance1, decimals1)
				tvl1Str = &val
			}
			tvlMethod = method
		}
	} else {
		tvlMethod = tvlMethodNone
	}

	feeRate0, feeRate1 := computeFeeRates(acc.Fee0, acc.Fee1, tvl0Int, tvl1Int)
	apr := computeAPR(feeRate0, feeRate1, a.cfg.WindowSeconds)

	metrics := &model.PoolWindowMetrics{
		ChainID:        acc.ChainID,
		PoolAddress:    acc.PoolAddress,
		WindowSizeSecs: int64(a.cfg.WindowSeconds),
		WindowStart:    time.Unix(int64(acc.WindowStart), 0).UTC(),
		WindowEnd:      time.Unix(int64(acc.WindowEnd), 0).UTC(),
		SwapCount:      acc.SwapCount,
		Volume0:        volume0,
		Volume1:        volume1,
		Fee0:           fee0,
		Fee1:           fee1,
		FeeUSD:         nil,
		FeeRate0:       feeRate0,
		FeeRate1:       feeRate1,
		TVL0:           tvl0Str,
		TVL1:           tvl1Str,
		TVLUSD:         nil,
		APR:            apr,
		FeeMethod:      feeMethodApprox,
		TVLMethod:      tvlMethod,
	}

	return metrics, poolRecord, nil
}

func (a *Aggregator) registerPool(acc *Accumulator) *model.Pool {
	key := poolKey(acc.PoolAddress)
	pool := model.Pool{
		ChainID:        acc.ChainID,
		Address:        acc.PoolAddress,
		Token0:         acc.PoolMeta.Token0,
		Token1:         acc.PoolMeta.Token1,
		Fee:            acc.PoolMeta.Fee,
		TickSpacing:    acc.PoolMeta.TickSpacing,
		FirstSeenBlock: acc.FirstBlock,
	}

	existing, ok := a.poolSeen[key]
	if ok {
		if existing.FirstSeenBlock <= pool.FirstSeenBlock {
			return nil
		}
	}

	a.poolSeen[key] = pool
	return &pool
}

func (a *Aggregator) getTokenDecimals(ctx context.Context, token string) (uint8, error) {
	if !common.IsHexAddress(token) {
		return 0, fmt.Errorf("invalid token address: %s", token)
	}
	addr := common.HexToAddress(token)
	if decimals, ok := a.decimals.Get(addr); ok {
		return decimals, nil
	}
	meta, err := FetchTokenDecimals(ctx, a.chainClient, addr)
	if err != nil {
		return 0, err
	}
	a.decimals.Set(addr, meta)
	return meta, nil
}

func windowStart(ts uint64, windowSec uint64) uint64 {
	return ts - (ts % windowSec)
}

func poolKey(address string) string {
	return strings.ToLower(address)
}

func minOpenWindowStart(acc map[string]*Accumulator) uint64 {
	var min uint64
	for _, entry := range acc {
		if entry == nil {
			continue
		}
		if min == 0 || entry.WindowStart < min {
			min = entry.WindowStart
		}
	}
	return min
}
