package aggregate

import (
	"math/big"
	"time"
)

const ratioScale = 18

func formatTokenAmount(value *big.Int, decimals uint8) string {
	if value == nil {
		return "0"
	}
	if decimals == 0 {
		return value.String()
	}
	sign := value.Sign()
	abs := new(big.Int).Abs(value)
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	rat := new(big.Rat).SetFrac(abs, denom)
	text := rat.FloatString(int(decimals))
	if sign < 0 {
		return "-" + text
	}
	return text
}

func computeFeeRates(fee0 *big.Int, fee1 *big.Int, tvl0 *big.Int, tvl1 *big.Int) (*string, *string) {
	var feeRate0 *string
	var feeRate1 *string

	if rate := computeRateFromInt(fee0, tvl0); rate != "" {
		feeRate0 = &rate
	}
	if rate := computeRateFromInt(fee1, tvl1); rate != "" {
		feeRate1 = &rate
	}
	return feeRate0, feeRate1
}

func computeRateFromInt(fee *big.Int, tvl *big.Int) string {
	if fee == nil || fee.Sign() == 0 || tvl == nil || tvl.Sign() == 0 {
		return ""
	}
	rat := new(big.Rat).SetFrac(fee, tvl)
	return rat.FloatString(ratioScale)
}

func computeAPR(feeRate0 *string, feeRate1 *string, windowSeconds uint64) *string {
	if windowSeconds == 0 {
		return nil
	}
	var selected string
	if feeRate0 != nil && feeRate1 == nil {
		selected = *feeRate0
	} else if feeRate1 != nil && feeRate0 == nil {
		selected = *feeRate1
	} else {
		return nil
	}

	rat, ok := new(big.Rat).SetString(selected)
	if !ok {
		return nil
	}
	yearSeconds := big.NewRat(int64(365*24*time.Hour/time.Second), 1)
	window := big.NewRat(int64(windowSeconds), 1)
	apr := new(big.Rat).Mul(rat, yearSeconds)
	apr.Quo(apr, window)
	val := apr.FloatString(ratioScale)
	return &val
}
