package model

// TypedEvent is a decoded pool event enriched with metadata.
type TypedEvent struct {
	ChainID     uint64      `json:"chain_id"`
	BlockNumber uint64      `json:"block_number"`
	BlockHash   string      `json:"block_hash"`
	TxHash      string      `json:"tx_hash"`
	LogIndex    uint64      `json:"log_index"`
	Address     string      `json:"address"`
	EventName   string      `json:"event_name"`
	Timestamp   uint64      `json:"timestamp"`
	Decoded     interface{} `json:"decoded"`
	PoolMeta    PoolMeta    `json:"pool_meta"`
	Raw         *RawLogRef  `json:"raw,omitempty"`
}

// RawLogRef keeps a minimal raw reference for traceability.
type RawLogRef struct {
	Topic0 string `json:"topic0"`
	Data   string `json:"data"`
}
