package vo

import "github.com/shopspring/decimal"

// CoinMarketDataVo is what a market data source knows about one coin; a nil figure means the source does not know it.
type CoinMarketDataVo struct {
	SourceName               string
	CoinGeckoID              string
	Name                     string
	FullyDilutedValuationUsd *decimal.Decimal
	DailyVolumeUsd           *decimal.Decimal
	CirculatingSupply        *decimal.Decimal
	TotalSupply              *decimal.Decimal
	MaxSupply                *decimal.Decimal
	ContractAddresses        []TokenAddressVo
}
