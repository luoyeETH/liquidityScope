package model

import "time"

// PoolWindowMetrics stores aggregated metrics for a pool window.
type PoolWindowMetrics struct {
	ChainID          uint64
	PoolAddress      string
	WindowSizeSecs   int64
	WindowStart      time.Time
	WindowEnd        time.Time
	SwapCount        uint64
	Volume0          string
	Volume1          string
	Fee0             string
	Fee1             string
	FeeUSD           *string
	FeeRate0         *string
	FeeRate1         *string
	TVL0             *string
	TVL1             *string
	TVLUSD           *string
	APR              *string
	FeeMethod        string
	TVLMethod        string
}
