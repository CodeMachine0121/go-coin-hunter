package entities

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/shopspring/decimal"
)

// CoinVerdict is the strategist's normalized verdict on one coin in one verdict run; ratios are fractions (0.1 is 10%).
type CoinVerdict struct {
	ID                 uint             `gorm:"primaryKey"`
	PipelineRunID      uint             `gorm:"not null;uniqueIndex:idx_coin_verdicts_run_symbol"`
	CoinSymbol         string           `gorm:"size:64;not null;uniqueIndex:idx_coin_verdicts_run_symbol"`
	Action             string           `gorm:"size:16;not null"`
	Confidence         int              `gorm:"not null"`
	Leverage           int              `gorm:"not null"`
	PositionSizeRatio  decimal.Decimal  `gorm:"type:numeric;not null"`
	ReferencePrice     *decimal.Decimal `gorm:"type:numeric"`
	StopLossRatio      *decimal.Decimal `gorm:"type:numeric"`
	StopLossPrice      *decimal.Decimal `gorm:"type:numeric"`
	TakeProfitRatio    *decimal.Decimal `gorm:"type:numeric"`
	TakeProfitPrice    *decimal.Decimal `gorm:"type:numeric"`
	Rationale          string
	ConflictResolution string
}

func (coinVerdict CoinVerdict) ToDto() dto.CoinVerdictDto {
	return dto.CoinVerdictDto{
		PipelineRunID:      coinVerdict.PipelineRunID,
		CoinSymbol:         coinVerdict.CoinSymbol,
		Action:             coinVerdict.Action,
		Confidence:         coinVerdict.Confidence,
		Leverage:           coinVerdict.Leverage,
		PositionSizeRatio:  coinVerdict.PositionSizeRatio,
		ReferencePrice:     coinVerdict.ReferencePrice,
		StopLossRatio:      coinVerdict.StopLossRatio,
		StopLossPrice:      coinVerdict.StopLossPrice,
		TakeProfitRatio:    coinVerdict.TakeProfitRatio,
		TakeProfitPrice:    coinVerdict.TakeProfitPrice,
		Rationale:          coinVerdict.Rationale,
		ConflictResolution: coinVerdict.ConflictResolution,
	}
}

// ToHuntBoardEntry is this verdict as the coin's row on the hunt board, calculated at the round's time.
func (coinVerdict CoinVerdict) ToHuntBoardEntry(calculatedAt time.Time) HuntBoardEntry {
	return HuntBoardEntry{
		CoinSymbol:         coinVerdict.CoinSymbol,
		CalculatedAt:       calculatedAt,
		PipelineRunID:      coinVerdict.PipelineRunID,
		Action:             coinVerdict.Action,
		Confidence:         coinVerdict.Confidence,
		Leverage:           coinVerdict.Leverage,
		PositionSizeRatio:  coinVerdict.PositionSizeRatio,
		ReferencePrice:     coinVerdict.ReferencePrice,
		StopLossRatio:      coinVerdict.StopLossRatio,
		StopLossPrice:      coinVerdict.StopLossPrice,
		TakeProfitRatio:    coinVerdict.TakeProfitRatio,
		TakeProfitPrice:    coinVerdict.TakeProfitPrice,
		Rationale:          coinVerdict.Rationale,
		ConflictResolution: coinVerdict.ConflictResolution,
	}
}
