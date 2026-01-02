package dex

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	"liquidityScope/internal/chain"
	"liquidityScope/internal/model"
)

// PoolMetaCache caches pool metadata by address.
type PoolMetaCache struct {
	mu   sync.RWMutex
	data map[common.Address]model.PoolMeta
}

func NewPoolMetaCache() *PoolMetaCache {
	return &PoolMetaCache{data: make(map[common.Address]model.PoolMeta)}
}

func (c *PoolMetaCache) Get(address common.Address) (model.PoolMeta, bool) {
	c.mu.RLock()
	meta, ok := c.data[address]
	c.mu.RUnlock()
	return meta, ok
}

func (c *PoolMetaCache) Set(address common.Address, meta model.PoolMeta) {
	c.mu.Lock()
	c.data[address] = meta
	c.mu.Unlock()
}

// TokenMetaCache caches token metadata by address.
type TokenMetaCache struct {
	mu   sync.RWMutex
	data map[common.Address]model.TokenMeta
}

func NewTokenMetaCache() *TokenMetaCache {
	return &TokenMetaCache{data: make(map[common.Address]model.TokenMeta)}
}

func (c *TokenMetaCache) Get(address common.Address) (model.TokenMeta, bool) {
	c.mu.RLock()
	meta, ok := c.data[address]
	c.mu.RUnlock()
	return meta, ok
}

func (c *TokenMetaCache) Set(address common.Address, meta model.TokenMeta) {
	c.mu.Lock()
	c.data[address] = meta
	c.mu.Unlock()
}

// FetchPoolMeta loads immutable pool metadata from chain and token caches.
func FetchPoolMeta(ctx context.Context, chainClient *chain.Client, pool common.Address, tokenCache *TokenMetaCache, logger *zap.Logger) (model.PoolMeta, error) {
	if chainClient == nil {
		return model.PoolMeta{}, fmt.Errorf("chain client is nil")
	}

	poolABI, err := V3PoolABI()
	if err != nil {
		return model.PoolMeta{}, fmt.Errorf("parse pool abi: %w", err)
	}

	values, err := callPoolMethod(ctx, chainClient, pool, poolABI, "token0", nil)
	if err != nil {
		return model.PoolMeta{}, err
	}
	token0, err := asAddress(values[0])
	if err != nil {
		return model.PoolMeta{}, fmt.Errorf("token0: %w", err)
	}

	values, err = callPoolMethod(ctx, chainClient, pool, poolABI, "token1", nil)
	if err != nil {
		return model.PoolMeta{}, err
	}
	token1, err := asAddress(values[0])
	if err != nil {
		return model.PoolMeta{}, fmt.Errorf("token1: %w", err)
	}

	values, err = callPoolMethod(ctx, chainClient, pool, poolABI, "fee", nil)
	if err != nil {
		return model.PoolMeta{}, err
	}
	feeInt, err := asBigInt(values[0])
	if err != nil {
		return model.PoolMeta{}, fmt.Errorf("fee: %w", err)
	}
	fee := uint32(feeInt.Uint64())

	values, err = callPoolMethod(ctx, chainClient, pool, poolABI, "tickSpacing", nil)
	if err != nil {
		return model.PoolMeta{}, err
	}
	tickSpacingInt, err := asBigInt(values[0])
	if err != nil {
		return model.PoolMeta{}, fmt.Errorf("tick spacing: %w", err)
	}
	tickSpacing, err := int24FromBig(tickSpacingInt)
	if err != nil {
		return model.PoolMeta{}, fmt.Errorf("tick spacing: %w", err)
	}

	meta := model.PoolMeta{
		Token0:      token0.Hex(),
		Token1:      token1.Hex(),
		Fee:         fee,
		TickSpacing: tickSpacing,
	}

	if tokenCache != nil {
		log := logger
		if log == nil {
			log = zap.NewNop()
		}
		if _, ok := tokenCache.Get(token0); !ok {
			if tokenMeta, err := FetchTokenMeta(ctx, chainClient, token0, log); err == nil {
				tokenCache.Set(token0, tokenMeta)
			} else {
				log.Warn("token0 metadata fetch failed", zap.String("token", token0.Hex()), zap.Error(err))
				tokenCache.Set(token0, tokenMeta)
			}
		}
		if _, ok := tokenCache.Get(token1); !ok {
			if tokenMeta, err := FetchTokenMeta(ctx, chainClient, token1, log); err == nil {
				tokenCache.Set(token1, tokenMeta)
			} else {
				log.Warn("token1 metadata fetch failed", zap.String("token", token1.Hex()), zap.Error(err))
				tokenCache.Set(token1, tokenMeta)
			}
		}
	}

	return meta, nil
}

// FetchPoolOptionalMeta loads optional pool fields (slot0/liquidity) at a block height.
func FetchPoolOptionalMeta(ctx context.Context, chainClient *chain.Client, pool common.Address, blockNumber uint64, logger *zap.Logger) (model.PoolMeta, error) {
	if chainClient == nil {
		return model.PoolMeta{}, fmt.Errorf("chain client is nil")
	}

	poolABI, err := V3PoolABI()
	if err != nil {
		return model.PoolMeta{}, fmt.Errorf("parse pool abi: %w", err)
	}

	var blockPtr *big.Int
	if blockNumber > 0 {
		blockPtr = new(big.Int).SetUint64(blockNumber)
	}

	meta := model.PoolMeta{}

	if values, err := callPoolMethod(ctx, chainClient, pool, poolABI, "liquidity", blockPtr); err == nil {
		if liq, err := asBigInt(values[0]); err == nil {
			meta.Liquidity = liq.String()
		}
	} else if logger != nil {
		logger.Debug("liquidity call failed", zap.String("pool", pool.Hex()), zap.Error(err))
	}

	if values, err := callPoolMethod(ctx, chainClient, pool, poolABI, "slot0", blockPtr); err == nil && len(values) >= 2 {
		sqrt, errSqrt := asBigInt(values[0])
		tickInt, errTick := asBigInt(values[1])
		if errSqrt == nil && errTick == nil {
			if tick, err := int24FromBig(tickInt); err == nil {
				meta.Slot0 = &model.PoolSlot0{
					SqrtPriceX96: sqrt.String(),
					Tick:         tick,
				}
			}
		}
	} else if err != nil && logger != nil {
		logger.Debug("slot0 call failed", zap.String("pool", pool.Hex()), zap.Error(err))
	}

	return meta, nil
}

func callPoolMethod(ctx context.Context, chainClient *chain.Client, pool common.Address, poolABI abi.ABI, method string, block *big.Int) ([]interface{}, error) {
	data, err := poolABI.Pack(method)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", method, err)
	}
	msg := ethereum.CallMsg{To: &pool, Data: data}
	resp, err := chainClient.CallContract(ctx, msg, block)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", method, err)
	}
	values, err := poolABI.Unpack(method, resp)
	if err != nil {
		return nil, fmt.Errorf("unpack %s: %w", method, err)
	}
	return values, nil
}

// FetchTokenMeta loads token metadata via ERC20 calls.
func FetchTokenMeta(ctx context.Context, chainClient *chain.Client, token common.Address, logger *zap.Logger) (model.TokenMeta, error) {
	meta := model.TokenMeta{Address: token.Hex()}
	if chainClient == nil {
		return meta, fmt.Errorf("chain client is nil")
	}

	stringABI, err := erc20ABIStringInstance()
	if err != nil {
		return meta, fmt.Errorf("parse erc20 string abi: %w", err)
	}
	bytes32ABI, err := erc20ABIBytes32Instance()
	if err != nil {
		return meta, fmt.Errorf("parse erc20 bytes32 abi: %w", err)
	}

	call := func(method string, parsed abi.ABI) ([]interface{}, error) {
		data, err := parsed.Pack(method)
		if err != nil {
			return nil, fmt.Errorf("pack %s: %w", method, err)
		}
		msg := ethereum.CallMsg{To: &token, Data: data}
		resp, err := chainClient.CallContract(ctx, msg, nil)
		if err != nil {
			return nil, fmt.Errorf("call %s: %w", method, err)
		}
		values, err := parsed.Unpack(method, resp)
		if err != nil {
			return nil, fmt.Errorf("unpack %s: %w", method, err)
		}
		return values, nil
	}

	values, err := call("decimals", stringABI)
	if err != nil {
		return meta, err
	}
	decimals, err := asUint8(values[0])
	if err != nil {
		return meta, err
	}
	meta.Decimals = decimals

	if values, err := call("symbol", stringABI); err == nil {
		if symbol, ok := values[0].(string); ok {
			meta.Symbol = symbol
		}
	} else if values, err := call("symbol", bytes32ABI); err == nil {
		if symbol, ok := bytes32ToString(values[0]); ok {
			meta.Symbol = symbol
		}
	} else if logger != nil {
		logger.Debug("symbol call failed", zap.String("token", token.Hex()), zap.Error(err))
	}

	if values, err := call("name", stringABI); err == nil {
		if name, ok := values[0].(string); ok {
			meta.Name = name
		}
	} else if values, err := call("name", bytes32ABI); err == nil {
		if name, ok := bytes32ToString(values[0]); ok {
			meta.Name = name
		}
	} else if logger != nil {
		logger.Debug("name call failed", zap.String("token", token.Hex()), zap.Error(err))
	}

	return meta, nil
}

func bytes32ToString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case [32]byte:
		return string(bytes.TrimRight(v[:], "\x00")), true
	case []byte:
		return string(bytes.TrimRight(v, "\x00")), true
	default:
		return "", false
	}
}

func asAddress(value interface{}) (common.Address, error) {
	switch v := value.(type) {
	case common.Address:
		return v, nil
	case *common.Address:
		return *v, nil
	default:
		return common.Address{}, fmt.Errorf("unsupported address type %T", value)
	}
}

func asBigInt(value interface{}) (*big.Int, error) {
	switch v := value.(type) {
	case *big.Int:
		return new(big.Int).Set(v), nil
	case big.Int:
		return new(big.Int).Set(&v), nil
	case uint8:
		return new(big.Int).SetUint64(uint64(v)), nil
	case uint16:
		return new(big.Int).SetUint64(uint64(v)), nil
	case uint32:
		return new(big.Int).SetUint64(uint64(v)), nil
	case uint64:
		return new(big.Int).SetUint64(v), nil
	case int8:
		return big.NewInt(int64(v)), nil
	case int16:
		return big.NewInt(int64(v)), nil
	case int32:
		return big.NewInt(int64(v)), nil
	case int64:
		return big.NewInt(v), nil
	default:
		return nil, fmt.Errorf("unsupported int type %T", value)
	}
}

func asUint8(value interface{}) (uint8, error) {
	switch v := value.(type) {
	case uint8:
		return v, nil
	case uint16:
		return uint8(v), nil
	case uint32:
		return uint8(v), nil
	case uint64:
		return uint8(v), nil
	case *big.Int:
		return uint8(v.Uint64()), nil
	default:
		return 0, fmt.Errorf("unsupported uint8 type %T", value)
	}
}

func int24FromBig(value *big.Int) (int32, error) {
	min := big.NewInt(-1 << 23)
	max := big.NewInt((1 << 23) - 1)
	if value.Cmp(min) < 0 || value.Cmp(max) > 0 {
		return 0, fmt.Errorf("int24 overflow: %s", value.String())
	}
	return int32(value.Int64()), nil
}
