package verdict

import (
	"encoding/json"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

type huntVerdictMaterialWire struct {
	CoinSymbol      string               `json:"coinSymbol"`
	Direction       string               `json:"direction"`
	Strength        int                  `json:"strength"`
	Catalyst        string               `json:"catalyst"`
	Risks           []string             `json:"risks"`
	Evidence        []string             `json:"evidence"`
	DataGaps        []string             `json:"dataGaps"`
	MarketStructure *marketStructureWire `json:"marketStructure"`
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

func newHuntVerdictMaterialWires(materials []vo.HuntVerdictMaterialVo) []huntVerdictMaterialWire {
	decimalText := func(value *decimal.Decimal) *string {
		if value == nil {
			return nil
		}
		text := value.String()
		return &text
	}
	wires := make([]huntVerdictMaterialWire, 0, len(materials))
	for _, material := range materials {
		wire := huntVerdictMaterialWire{CoinSymbol: material.CoinSymbol, Direction: material.Direction, Strength: material.Strength,
			Catalyst: material.Catalyst, Risks: append([]string{}, material.Risks...), Evidence: append([]string{}, material.Evidence...),
			DataGaps: append([]string{}, material.DataGaps...)}
		if marketStructure := material.MarketStructure; marketStructure != nil {
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
		wires = append(wires, wire)
	}

	return wires
}

type huntVerdictAnswerWire struct {
	Verdicts *[]huntVerdictWire `json:"verdicts"`
}

// huntVerdictWire mirrors one verdict of the answer schema; pointers tell a missing field from a zero one.
type huntVerdictWire struct {
	CoinSymbol          *string      `json:"coinSymbol"`
	Action              *string      `json:"action"`
	Confidence          *json.Number `json:"confidence"`
	Leverage            *json.Number `json:"leverage"`
	PositionSizePercent *json.Number `json:"positionSizePercent"`
	StopLossPercent     *json.Number `json:"stopLossPercent"`
	TakeProfitPercent   *json.Number `json:"takeProfitPercent"`
	Rationale           *string      `json:"rationale"`
	ConflictResolution  *string      `json:"conflictResolution"`
}

// toHuntVerdictAnswer reads numbers exactly; whole-number fields round, and any missing field is unreadable.
func (verdict huntVerdictWire) toHuntVerdictAnswer() (vo.HuntVerdictAnswerVo, bool) {
	if verdict.CoinSymbol == nil || verdict.Action == nil || verdict.Rationale == nil || verdict.ConflictResolution == nil {
		return vo.HuntVerdictAnswerVo{}, false
	}
	numbers := []*json.Number{verdict.Confidence, verdict.Leverage, verdict.PositionSizePercent, verdict.StopLossPercent, verdict.TakeProfitPercent}
	values := make([]decimal.Decimal, 0, len(numbers))
	for _, number := range numbers {
		if number == nil {
			return vo.HuntVerdictAnswerVo{}, false
		}
		// A json.Number the decoder accepted is always a valid decimal.
		value, _ := decimal.NewFromString(number.String())
		values = append(values, value)
	}

	return vo.HuntVerdictAnswerVo{
		CoinSymbol:          *verdict.CoinSymbol,
		Action:              *verdict.Action,
		Confidence:          int(values[0].Round(0).IntPart()),
		Leverage:            int(values[1].Round(0).IntPart()),
		PositionSizePercent: values[2],
		StopLossPercent:     values[3],
		TakeProfitPercent:   values[4],
		Rationale:           *verdict.Rationale,
		ConflictResolution:  *verdict.ConflictResolution,
	}, true
}
