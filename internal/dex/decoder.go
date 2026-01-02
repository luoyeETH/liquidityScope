package dex

import (
	"context"

	"go.uber.org/zap"

	"liquidityScope/internal/chain"
	"liquidityScope/internal/model"
)

// Decoder defines a log decoder.
type Decoder interface {
	CanDecode(topic0 string) bool
	Decode(log model.LogRecord, ctx DecodeContext) (*model.TypedEvent, error)
}

// DecodeContext provides shared dependencies for decoders.
type DecodeContext struct {
	Context         context.Context
	Chain           *chain.Client
	PoolMetaCache   *PoolMetaCache
	TokenMetaCache  *TokenMetaCache
	Logger          *zap.Logger
	IncludeLiveMeta bool
}
