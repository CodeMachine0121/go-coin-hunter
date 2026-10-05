package insight

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// coinInsightMaterialWire is the material as the analyst reads it: plain JSON with the field names the prompt explains.
type coinInsightMaterialWire struct {
	CoinSymbol            string               `json:"coinSymbol"`
	IntelligenceHeadlines []headlineWire       `json:"intelligenceHeadlines"`
	NewsHeadlines         []headlineWire       `json:"newsHeadlines"`
	MarketStructure       *marketStructureWire `json:"marketStructure"`
	FilterVerdicts        []filterVerdictWire  `json:"filterVerdicts"`
	DataGaps              []string             `json:"dataGaps"`
}

type headlineWire struct {
	Source      string    `json:"source"`
	Title       string    `json:"title"`
	PublishedAt time.Time `json:"publishedAt"`
}

// marketStructureWire keeps figures as exact decimal text; an unknown figure is null.
type marketStructureWire struct {
	Exchange                   string  `json:"exchange"`
	LastPrice                  *string `json:"lastPrice"`
	PriceChangeRatio24h        *string `json:"priceChangeRatio24h"`
	QuoteVolumeUsd24h          *string `json:"quoteVolumeUsd24h"`
	FundingRate                *string `json:"fundingRate"`
	OpenInterestUsd            *string `json:"openInterestUsd"`
	OpenInterestChangeRatio24h *string `json:"openInterestChangeRatio24h"`
}

type filterVerdictWire struct {
	Rule    string `json:"rule"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}

func newCoinInsightMaterialWire(material vo.CoinInsightMaterialVo) coinInsightMaterialWire {
	wire := coinInsightMaterialWire{
		CoinSymbol:            material.CoinSymbol,
		IntelligenceHeadlines: []headlineWire{},
		NewsHeadlines:         []headlineWire{},
		FilterVerdicts:        []filterVerdictWire{},
		DataGaps:              append([]string{}, material.DataGaps...),
	}
	for _, headline := range material.IntelligenceHeadlines {
		wire.IntelligenceHeadlines = append(wire.IntelligenceHeadlines, headlineWire{Source: headline.SourceName, Title: headline.Title, PublishedAt: headline.PublishedAt})
	}
	for _, headline := range material.NewsHeadlines {
		wire.NewsHeadlines = append(wire.NewsHeadlines, headlineWire{Source: headline.SourceName, Title: headline.Title, PublishedAt: headline.PublishedAt})
	}
	for _, verdict := range material.FilterVerdicts {
		wire.FilterVerdicts = append(wire.FilterVerdicts, filterVerdictWire{Rule: verdict.FilterName, Outcome: string(verdict.Outcome), Reason: verdict.Reason})
	}
	if marketStructure := material.MarketStructure; marketStructure != nil {
		decimalText := func(value *decimal.Decimal) *string {
			if value == nil {
				return nil
			}
			text := value.String()
			return &text
		}
		wire.MarketStructure = &marketStructureWire{
			Exchange:                   marketStructure.ExchangeName,
			LastPrice:                  decimalText(marketStructure.LastPrice),
			PriceChangeRatio24h:        decimalText(marketStructure.PriceChangeRatio24h),
			QuoteVolumeUsd24h:          decimalText(marketStructure.QuoteVolumeUsd24h),
			FundingRate:                decimalText(marketStructure.FundingRate),
			OpenInterestUsd:            decimalText(marketStructure.OpenInterestUsd),
			OpenInterestChangeRatio24h: decimalText(marketStructure.OpenInterestChangeRatio24h),
		}
	}

	return wire
}

// coinInsightAnswerWire mirrors the answer schema; pointers tell a missing field from an empty one.
type coinInsightAnswerWire struct {
	Direction *string   `json:"direction"`
	Strength  *int      `json:"strength"`
	Catalyst  *string   `json:"catalyst"`
	Risks     *[]string `json:"risks"`
	Evidence  *[]string `json:"evidence"`
	DataGaps  *[]string `json:"dataGaps"`
}

func (answer coinInsightAnswerWire) complete() bool {
	return answer.Direction != nil && answer.Strength != nil && answer.Catalyst != nil &&
		answer.Risks != nil && answer.Evidence != nil && answer.DataGaps != nil
}

func (answer coinInsightAnswerWire) toCoinInsightAnswer() vo.CoinInsightAnswerVo {
	return vo.CoinInsightAnswerVo{
		Direction: *answer.Direction,
		Strength:  *answer.Strength,
		Catalyst:  *answer.Catalyst,
		Risks:     *answer.Risks,
		Evidence:  *answer.Evidence,
		DataGaps:  *answer.DataGaps,
	}
}
