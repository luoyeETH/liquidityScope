package indexer

import (
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"liquidityScope/internal/model"
)

func buildLogRecord(chainID uint64, log types.Log, timestamp uint64, ingestedAt time.Time) model.LogRecord {
	topics := make([]string, 0, len(log.Topics))
	for _, topic := range log.Topics {
		topics = append(topics, topic.Hex())
	}

	return model.LogRecord{
		ChainID:     chainID,
		BlockNumber: log.BlockNumber,
		BlockHash:   log.BlockHash.Hex(),
		TxHash:      log.TxHash.Hex(),
		TxIndex:     uint64(log.TxIndex),
		LogIndex:    uint64(log.Index),
		Address:     log.Address.Hex(),
		Topics:      topics,
		Data:        hexutil.Encode(log.Data),
		Removed:     log.Removed,
		Timestamp:   timestamp,
		IngestedAt:  ingestedAt.UTC().Format(time.RFC3339Nano),
	}
}
