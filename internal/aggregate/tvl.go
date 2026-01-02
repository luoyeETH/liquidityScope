package aggregate

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"liquidityScope/internal/chain"
)

const erc20BalanceOfABIJSON = `[
  {"inputs": [{"internalType": "address", "name": "account", "type": "address"}], "name": "balanceOf", "outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}], "stateMutability": "view", "type": "function"}
]`

var (
	balanceOfABI     abi.ABI
	balanceOfOnce    sync.Once
	balanceOfABIErr  error
)

func getBalanceOfABI() (abi.ABI, error) {
	balanceOfOnce.Do(func() {
		balanceOfABI, balanceOfABIErr = abi.JSON(strings.NewReader(erc20BalanceOfABIJSON))
	})
	return balanceOfABI, balanceOfABIErr
}

func (a *Aggregator) fetchTVL(ctx context.Context, token0, token1, poolAddr string, blockNumber uint64) (*big.Int, *big.Int, string, error) {
	if !common.IsHexAddress(token0) || !common.IsHexAddress(token1) || !common.IsHexAddress(poolAddr) {
		return nil, nil, tvlMethodNone, fmt.Errorf("invalid address")
	}

	pool := common.HexToAddress(poolAddr)
	blockPtr := new(big.Int).SetUint64(blockNumber)

	bal0, err0 := balanceOf(ctx, a.chainClient, common.HexToAddress(token0), pool, blockPtr)
	bal1, err1 := balanceOf(ctx, a.chainClient, common.HexToAddress(token1), pool, blockPtr)
	if err0 == nil && err1 == nil {
		return bal0, bal1, tvlMethodBlock, nil
	}

	bal0, err0 = balanceOf(ctx, a.chainClient, common.HexToAddress(token0), pool, nil)
	bal1, err1 = balanceOf(ctx, a.chainClient, common.HexToAddress(token1), pool, nil)
	if err0 == nil && err1 == nil {
		return bal0, bal1, tvlMethodLatest, nil
	}

	return nil, nil, tvlMethodNone, fmt.Errorf("balanceOf failed")
}

func balanceOf(ctx context.Context, chainClient *chain.Client, token common.Address, owner common.Address, blockNumber *big.Int) (*big.Int, error) {
	if chainClient == nil {
		return nil, fmt.Errorf("chain client is nil")
	}
	balanceABI, err := getBalanceOfABI()
	if err != nil {
		return nil, err
	}

	data, err := balanceABI.Pack("balanceOf", owner)
	if err != nil {
		return nil, fmt.Errorf("pack balanceOf: %w", err)
	}

	msg := ethereum.CallMsg{To: &token, Data: data}
	resp, err := chainClient.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("call balanceOf: %w", err)
	}

	values, err := balanceABI.Unpack("balanceOf", resp)
	if err != nil {
		return nil, fmt.Errorf("unpack balanceOf: %w", err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("balanceOf return size %d", len(values))
	}
	bal, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("balanceOf unexpected type %T", values[0])
	}
	return bal, nil
}
