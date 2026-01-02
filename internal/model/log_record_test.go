package model

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestLogRecordJSONRoundTrip(t *testing.T) {
	original := LogRecord{
		ChainID:     56,
		BlockNumber: 36000000,
		BlockHash:   "0xabc123",
		TxHash:      "0xdef456",
		TxIndex:     7,
		LogIndex:    12,
		Address:     "0x1111111111111111111111111111111111111111",
		Topics:      []string{"0xaaa", "0xbbb"},
		Data:        "0xdeadbeef",
		Removed:     false,
		Timestamp:   1700000000,
		IngestedAt:  "2024-01-01T00:00:00Z",
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded LogRecord
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch: %+v != %+v", original, decoded)
	}
}
