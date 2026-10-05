package vo

import (
	"time"

	"github.com/shopspring/decimal"
)

// CoinInsightMaterialVo is everything the analyst is shown about one candidate.
type CoinInsightMaterialVo struct {
	CoinSymbol            string
	IntelligenceHeadlines []HeadlineVo
	NewsHeadlines         []HeadlineVo
	MarketStructure       *PerpetualMarketStructureVo
	FilterVerdicts        []FilterVerdictVo
	// DataGaps names the material that could not be gathered, so the analyst and the trader know what is missing.
	DataGaps []string
}

// HeadlineVo is one titled item with where and when it appeared.
type HeadlineVo struct {
	SourceName  string
	Title       string
	PublishedAt time.Time
}

// PerpetualMarketStructureVo is one exchange's USDT perpetual for the coin; a nil figure is unknown there.
type PerpetualMarketStructureVo struct {
	ExchangeName               string
	LastPrice                  *decimal.Decimal
	PriceChangeRatio24h        *decimal.Decimal
	QuoteVolumeUsd24h          *decimal.Decimal
	FundingRate                *decimal.Decimal
	OpenInterestUsd            *decimal.Decimal
	OpenInterestChangeRatio24h *decimal.Decimal
}
