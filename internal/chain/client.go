package chain

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client wraps go-ethereum RPC and provides helper methods.
type Client struct {
	rpcClient *rpc.Client
	ethClient *ethclient.Client

	mu        sync.RWMutex
	tsCache   map[uint64]uint64
}

// NewClient creates a new chain client from the RPC URL.
func NewClient(ctx context.Context, rpcURL string) (*Client, error) {
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		rpcClient: rpcClient,
		ethClient: ethclient.NewClient(rpcClient),
		tsCache:   make(map[uint64]uint64),
	}, nil
}

// Close closes the underlying RPC client.
func (c *Client) Close() {
	if c.rpcClient != nil {
		c.rpcClient.Close()
	}
}

// GetChainID returns the chain ID.
func (c *Client) GetChainID(ctx context.Context) (*big.Int, error) {
	return c.ethClient.ChainID(ctx)
}

// LatestBlockNumber returns the latest block number.
func (c *Client) LatestBlockNumber(ctx context.Context) (uint64, error) {
	return c.ethClient.BlockNumber(ctx)
}

// BlockByNumber returns the block by number.
func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return c.ethClient.BlockByNumber(ctx, number)
}

// HeaderByNumber returns the block header by number.
func (c *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return c.ethClient.HeaderByNumber(ctx, number)
}

// BlockTimestamp returns the block timestamp, using an in-memory cache.
func (c *Client) BlockTimestamp(ctx context.Context, number uint64) (uint64, error) {
	c.mu.RLock()
	ts, ok := c.tsCache[number]
	c.mu.RUnlock()
	if ok {
		return ts, nil
	}

	header, err := c.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return 0, err
	}

	ts = header.Time
	c.mu.Lock()
	c.tsCache[number] = ts
	c.mu.Unlock()

	return ts, nil
}

// FilterLogs returns logs in the given range for addresses and topic0 filters.
func (c *Client) FilterLogs(
	ctx context.Context,
	fromBlock uint64,
	toBlock uint64,
	addresses []common.Address,
	topic0 []common.Hash,
) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: addresses,
	}
	if len(topic0) > 0 {
		query.Topics = [][]common.Hash{topic0}
	}
	return c.ethClient.FilterLogs(ctx, query)
}

// CallContract performs an eth_call for a contract method.
func (c *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return c.ethClient.CallContract(ctx, msg, blockNumber)
}
