package indexer

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ParseAddresses converts string addresses into common.Address.
func ParseAddresses(inputs []string) ([]common.Address, error) {
	addresses := make([]common.Address, 0, len(inputs))
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if !common.IsHexAddress(input) {
			return nil, fmt.Errorf("invalid address: %s", input)
		}
		addresses = append(addresses, common.HexToAddress(input))
	}
	return addresses, nil
}

// ParseTopic0 converts string topic0 hashes into common.Hash.
func ParseTopic0(inputs []string) ([]common.Hash, error) {
	topics := make([]common.Hash, 0, len(inputs))
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		data, err := hexutil.Decode(input)
		if err != nil {
			return nil, fmt.Errorf("invalid topic0: %s", input)
		}
		if len(data) != 32 {
			return nil, fmt.Errorf("invalid topic0 length: %s", input)
		}
		topics = append(topics, common.BytesToHash(data))
	}
	return topics, nil
}
