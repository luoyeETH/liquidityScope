package aggregate

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"liquidityScope/internal/chain"
	"liquidityScope/internal/dex"
)

// TokenDecimalsCache caches token decimals by address.
type TokenDecimalsCache struct {
	mu   sync.RWMutex
	data map[common.Address]uint8
}

func NewTokenDecimalsCache() *TokenDecimalsCache {
	return &TokenDecimalsCache{data: make(map[common.Address]uint8)}
}

func (c *TokenDecimalsCache) Get(address common.Address) (uint8, bool) {
	c.mu.RLock()
	decimals, ok := c.data[address]
	c.mu.RUnlock()
	return decimals, ok
}

func (c *TokenDecimalsCache) Set(address common.Address, decimals uint8) {
	c.mu.Lock()
	c.data[address] = decimals
	c.mu.Unlock()
}

// FetchTokenDecimals loads token decimals via chain RPC.
func FetchTokenDecimals(ctx context.Context, chainClient *chain.Client, token common.Address) (uint8, error) {
	if chainClient == nil {
		return 0, fmt.Errorf("chain client is nil")
	}
	meta, err := dex.FetchTokenMeta(ctx, chainClient, token, nil)
	if err != nil {
		return 0, err
	}
	return meta.Decimals, nil
}
