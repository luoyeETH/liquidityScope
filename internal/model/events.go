package model

// SwapEventData is the decoded Swap event payload.
type SwapEventData struct {
	Sender       string `json:"sender"`
	Recipient    string `json:"recipient"`
	Amount0      string `json:"amount0"`
	Amount1      string `json:"amount1"`
	SqrtPriceX96 string `json:"sqrt_price_x96"`
	Liquidity    string `json:"liquidity"`
	Tick         int32  `json:"tick"`
}

// MintEventData is the decoded Mint event payload.
type MintEventData struct {
	Sender    string `json:"sender"`
	Owner     string `json:"owner"`
	TickLower int32  `json:"tick_lower"`
	TickUpper int32  `json:"tick_upper"`
	Amount    string `json:"amount"`
	Amount0   string `json:"amount0"`
	Amount1   string `json:"amount1"`
}

// BurnEventData is the decoded Burn event payload.
type BurnEventData struct {
	Owner     string `json:"owner"`
	TickLower int32  `json:"tick_lower"`
	TickUpper int32  `json:"tick_upper"`
	Amount    string `json:"amount"`
	Amount0   string `json:"amount0"`
	Amount1   string `json:"amount1"`
}

// CollectEventData is the decoded Collect event payload.
type CollectEventData struct {
	Owner     string `json:"owner"`
	Recipient string `json:"recipient"`
	TickLower int32  `json:"tick_lower"`
	TickUpper int32  `json:"tick_upper"`
	Amount0   string `json:"amount0"`
	Amount1   string `json:"amount1"`
}
