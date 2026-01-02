package model

// Pool represents a V3 pool metadata record for storage.
type Pool struct {
	ChainID        uint64 `json:"chain_id"`
	Address        string `json:"address"`
	Token0         string `json:"token0"`
	Token1         string `json:"token1"`
	Fee            uint32 `json:"fee"`
	TickSpacing    int32  `json:"tick_spacing"`
	FirstSeenBlock uint64 `json:"first_seen_block"`
}
