package model

import (
	"encoding/json"
	"testing"
)

func TestSwapEventDataJSONStringFields(t *testing.T) {
	payload := SwapEventData{
		Sender:       "0x1111111111111111111111111111111111111111",
		Recipient:    "0x2222222222222222222222222222222222222222",
		Amount0:      "12345678901234567890",
		Amount1:      "-42",
		SqrtPriceX96: "79228162514264337593543950336",
		Liquidity:    "5000000000000000000",
		Tick:         10,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if _, ok := decoded["amount0"].(string); !ok {
		t.Fatalf("amount0 should be string")
	}
	if _, ok := decoded["amount1"].(string); !ok {
		t.Fatalf("amount1 should be string")
	}
	if _, ok := decoded["sqrt_price_x96"].(string); !ok {
		t.Fatalf("sqrt_price_x96 should be string")
	}
	if _, ok := decoded["liquidity"].(string); !ok {
		t.Fatalf("liquidity should be string")
	}
}
