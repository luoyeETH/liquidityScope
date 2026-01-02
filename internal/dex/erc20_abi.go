package dex

import (
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const erc20ABIStringJSON = `[
  {"inputs": [], "name": "decimals", "outputs": [{"type": "uint8"}], "stateMutability": "view", "type": "function"},
  {"inputs": [], "name": "symbol", "outputs": [{"type": "string"}], "stateMutability": "view", "type": "function"},
  {"inputs": [], "name": "name", "outputs": [{"type": "string"}], "stateMutability": "view", "type": "function"}
]`

const erc20ABIBytes32JSON = `[
  {"inputs": [], "name": "decimals", "outputs": [{"type": "uint8"}], "stateMutability": "view", "type": "function"},
  {"inputs": [], "name": "symbol", "outputs": [{"type": "bytes32"}], "stateMutability": "view", "type": "function"},
  {"inputs": [], "name": "name", "outputs": [{"type": "bytes32"}], "stateMutability": "view", "type": "function"}
]`

var (
	erc20ABIString     abi.ABI
	erc20ABIStringOnce sync.Once
	erc20ABIStringErr  error
	erc20ABIBytes32    abi.ABI
	erc20ABIBytes32Once sync.Once
	erc20ABIBytes32Err error
)

func erc20ABIStringInstance() (abi.ABI, error) {
	erc20ABIStringOnce.Do(func() {
		erc20ABIString, erc20ABIStringErr = abi.JSON(strings.NewReader(erc20ABIStringJSON))
	})
	return erc20ABIString, erc20ABIStringErr
}

func erc20ABIBytes32Instance() (abi.ABI, error) {
	erc20ABIBytes32Once.Do(func() {
		erc20ABIBytes32, erc20ABIBytes32Err = abi.JSON(strings.NewReader(erc20ABIBytes32JSON))
	})
	return erc20ABIBytes32, erc20ABIBytes32Err
}
