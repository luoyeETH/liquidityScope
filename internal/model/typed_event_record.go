package model

import "encoding/json"

// TypedEventRecord is the JSON representation used for aggregation.
type TypedEventRecord struct {
	ChainID     uint64          `json:"chain_id"`
	BlockNumber uint64          `json:"block_number"`
	BlockHash   string          `json:"block_hash"`
	TxHash      string          `json:"tx_hash"`
	LogIndex    uint64          `json:"log_index"`
	Address     string          `json:"address"`
	EventName   string          `json:"event_name"`
	Timestamp   uint64          `json:"timestamp"`
	Decoded     json.RawMessage `json:"decoded"`
	PoolMeta    PoolMeta        `json:"pool_meta"`
	Raw         *RawLogRef      `json:"raw,omitempty"`
}
