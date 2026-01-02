package model

// DecodeError records a decode failure for a log line.
type DecodeError struct {
	ChainID     uint64 `json:"chain_id"`
	BlockNumber uint64 `json:"block_number"`
	TxHash      string `json:"tx_hash"`
	LogIndex    uint64 `json:"log_index"`
	Address     string `json:"address"`
	Topic0      string `json:"topic0"`
	Error       string `json:"error"`
}
