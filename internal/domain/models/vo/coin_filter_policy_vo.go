package vo

import (
	"time"

	"github.com/shopspring/decimal"
)

// CoinFilterPolicyVo holds every operator-tunable filtering threshold; boundaries are inclusive as the rules state.
type CoinFilterPolicyVo struct {
	MaximumTaxRate                  decimal.Decimal
	MinimumDailyVolumeUsd           decimal.Decimal
	MinimumFullyDilutedValuationUsd decimal.Decimal
	MaximumFullyDilutedValuationUsd decimal.Decimal
	MinimumCirculatingRatio         decimal.Decimal
	UnlockLookahead                 time.Duration
	MaximumUnlockRatioOfCirculating decimal.Decimal
}
