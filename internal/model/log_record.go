package model

import (
	"encoding/json"
)

// LogRecord is the normalized representation of a chain log for storage.
type LogRecord struct {
	ChainID     uint64   `json:"chain_id"`
	BlockNumber uint64   `json:"block_number"`
	BlockHash   string   `json:"block_hash"`
	TxHash      string   `json:"tx_hash"`
	TxIndex     uint64   `json:"tx_index"`
	LogIndex    uint64   `json:"log_index"`
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	Removed     bool     `json:"removed"`
	Timestamp   uint64   `json:"timestamp"`
	IngestedAt  string   `json:"ingested_at"`
}

// MarshalJSON ensures LogRecord is encoded with stable field names.
func (lr LogRecord) MarshalJSON() ([]byte, error) {
	type Alias LogRecord
	return json.Marshal(Alias(lr))
}

// UnmarshalJSON decodes a LogRecord from JSON.
func (lr *LogRecord) UnmarshalJSON(data []byte) error {
	type Alias LogRecord
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*lr = LogRecord(a)
	return nil
}
