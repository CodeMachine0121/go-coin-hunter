package domains

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// BullishFocusDomain says how bullish is bullish enough: which insights the strategist weighs, and which of its verdicts
// make the hunt board. Everything else stays in the verdict history only.
type BullishFocusDomain struct {
	minimumBullishInsightStrength int
	minimumHuntBoardConfidence    int
}

// NewBullishFocusDomain keeps the thresholds on the scales they are compared with: a strength of 1–10 and a confidence of
// 0–100. A threshold off its scale would silently keep everything or nothing.
func NewBullishFocusDomain(huntVerdictPolicy vo.HuntVerdictPolicyVo) BullishFocusDomain {
	return BullishFocusDomain{
		minimumBullishInsightStrength: min(max(huntVerdictPolicy.MinimumBullishInsightStrength, minimumInsightStrength), maximumInsightStrength),
		minimumHuntBoardConfidence:    min(max(huntVerdictPolicy.MinimumHuntBoardConfidence, minimumVerdictConfidence), maximumVerdictConfidence),
	}
}

// SelectBullishInsights keeps, in order, the analyzed insights that are bullish and at least as strong as the threshold.
func (bullishFocusDomain BullishFocusDomain) SelectBullishInsights(coinInsights []entities.CoinInsight) []entities.CoinInsight {
	bullishInsights := []entities.CoinInsight{}
	for _, coinInsight := range coinInsights {
		if coinInsight.Succeeded && coinInsight.Direction == string(vo.CoinInsightDirectionBullish) &&
			coinInsight.Strength >= bullishFocusDomain.minimumBullishInsightStrength {
			bullishInsights = append(bullishInsights, coinInsight)
		}
	}

	return bullishInsights
}

// ToHuntBoardEntries puts on the board, in order, the long verdicts at least as confident as the threshold.
func (bullishFocusDomain BullishFocusDomain) ToHuntBoardEntries(coinVerdicts []entities.CoinVerdict, calculatedAt time.Time) []entities.HuntBoardEntry {
	huntBoardEntries := []entities.HuntBoardEntry{}
	for _, coinVerdict := range coinVerdicts {
		if coinVerdict.Action == string(vo.HuntActionLong) && coinVerdict.Confidence >= bullishFocusDomain.minimumHuntBoardConfidence {
			huntBoardEntries = append(huntBoardEntries, coinVerdict.ToHuntBoardEntry(calculatedAt))
		}
	}

	return huntBoardEntries
}
