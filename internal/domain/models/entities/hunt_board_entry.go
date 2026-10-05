package entities

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/shopspring/decimal"
)

// HuntBoardEntry is one coin's row on the hunt board: the latest successful verdict about it and when it was calculated.
type HuntBoardEntry struct {
	ID                 uint             `gorm:"primaryKey"`
	CoinSymbol         string           `gorm:"size:64;not null;uniqueIndex"`
	CalculatedAt       time.Time        `gorm:"not null"`
	PipelineRunID      uint             `gorm:"not null"`
	Action             string           `gorm:"size:16;not null"`
	Confidence         int              `gorm:"not null;index"`
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

func (huntBoardEntry HuntBoardEntry) ToDto() dto.HuntBoardEntryDto {
	return dto.HuntBoardEntryDto{
		ID:           huntBoardEntry.ID,
		CoinSymbol:   huntBoardEntry.CoinSymbol,
		CalculatedAt: huntBoardEntry.CalculatedAt,
		CoinVerdictDto: dto.CoinVerdictDto{
			PipelineRunID:      huntBoardEntry.PipelineRunID,
			CoinSymbol:         huntBoardEntry.CoinSymbol,
			Action:             huntBoardEntry.Action,
			Confidence:         huntBoardEntry.Confidence,
			Leverage:           huntBoardEntry.Leverage,
			PositionSizeRatio:  huntBoardEntry.PositionSizeRatio,
			ReferencePrice:     huntBoardEntry.ReferencePrice,
			StopLossRatio:      huntBoardEntry.StopLossRatio,
			StopLossPrice:      huntBoardEntry.StopLossPrice,
			TakeProfitRatio:    huntBoardEntry.TakeProfitRatio,
			TakeProfitPrice:    huntBoardEntry.TakeProfitPrice,
			Rationale:          huntBoardEntry.Rationale,
			ConflictResolution: huntBoardEntry.ConflictResolution,
		},
	}
}
