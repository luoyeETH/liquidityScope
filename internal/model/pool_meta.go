package model

// PoolMeta captures immutable pool metadata with optional live fields.
type PoolMeta struct {
	Token0      string     `json:"token0"`
	Token1      string     `json:"token1"`
	Fee         uint32     `json:"fee"`
	TickSpacing int32      `json:"tick_spacing"`
	Liquidity   string     `json:"liquidity,omitempty"`
	Slot0       *PoolSlot0 `json:"slot0,omitempty"`
}

// PoolSlot0 includes select slot0 fields.
type PoolSlot0 struct {
	SqrtPriceX96 string `json:"sqrt_price_x96"`
	Tick         int32  `json:"tick"`
}
