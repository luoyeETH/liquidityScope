package model

// TokenMeta captures ERC20 metadata.
type TokenMeta struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
}
