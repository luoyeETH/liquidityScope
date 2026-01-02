package aggregate

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"liquidityScope/internal/model"
)

// Accumulator holds aggregate values for a pool window.
type Accumulator struct {
	ChainID     uint64
	PoolAddress string
	PoolMeta    model.PoolMeta
	WindowStart uint64
	WindowEnd   uint64
	SwapCount   uint64
	Volume0     *big.Int
	Volume1     *big.Int
	Fee0        *big.Int
	Fee1        *big.Int
	LastBlock   uint64
	LastTS      uint64
	FirstBlock  uint64
}

func NewAccumulator(record model.TypedEventRecord, windowStart, windowEnd uint64) *Accumulator {
	return &Accumulator{
		ChainID:     record.ChainID,
		PoolAddress: record.Address,
		PoolMeta:    record.PoolMeta,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Volume0:     big.NewInt(0),
		Volume1:     big.NewInt(0),
		Fee0:        big.NewInt(0),
		Fee1:        big.NewInt(0),
		LastBlock:   record.BlockNumber,
		LastTS:      record.Timestamp,
		FirstBlock:  record.BlockNumber,
	}
}

func (a *Accumulator) AddEvent(record model.TypedEventRecord) error {
	if record.Timestamp >= a.LastTS {
		a.LastTS = record.Timestamp
		a.LastBlock = record.BlockNumber
	}
	if a.FirstBlock == 0 || record.BlockNumber < a.FirstBlock {
		a.FirstBlock = record.BlockNumber
	}

	switch strings.ToLower(record.EventName) {
	case "swap":
		var swap model.SwapEventData
		if err := json.Unmarshal(record.Decoded, &swap); err != nil {
			return fmt.Errorf("decode swap: %w", err)
		}
		return a.applySwap(swap)
	default:
		return nil
	}
}

func (a *Accumulator) applySwap(swap model.SwapEventData) error {
	amount0, err := parseBigInt(swap.Amount0)
	if err != nil {
		return err
	}
	amount1, err := parseBigInt(swap.Amount1)
	if err != nil {
		return err
	}

	absAdd(a.Volume0, amount0)
	absAdd(a.Volume1, amount1)
	feeRate := a.PoolMeta.Fee
	if feeRate == 0 {
		a.SwapCount++
		return nil
	}

	if amount0.Sign() < 0 && amount1.Sign() > 0 {
		fee := feeFromAmount(amount1, feeRate)
		a.Fee1.Add(a.Fee1, fee)
	} else if amount1.Sign() < 0 && amount0.Sign() > 0 {
		fee := feeFromAmount(amount0, feeRate)
		a.Fee0.Add(a.Fee0, fee)
	}

	a.SwapCount++
	return nil
}

func parseBigInt(value string) (*big.Int, error) {
	if value == "" {
		return big.NewInt(0), nil
	}
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("invalid int: %s", value)
	}
	return parsed, nil
}

func absAdd(target *big.Int, value *big.Int) {
	if value == nil || target == nil {
		return
	}
	abs := new(big.Int).Abs(value)
	target.Add(target, abs)
}

func feeFromAmount(amountIn *big.Int, feeRate uint32) *big.Int {
	if amountIn == nil {
		return big.NewInt(0)
	}
	fee := new(big.Int).Abs(amountIn)
	fee.Mul(fee, big.NewInt(int64(feeRate)))
	fee.Div(fee, big.NewInt(1_000_000))
	return fee
}
