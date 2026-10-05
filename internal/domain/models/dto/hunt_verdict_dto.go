package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

// CoinVerdictDto ratios are fractions (0.1 is 10%); prices are absent for watch and avoid.
type CoinVerdictDto struct {
	PipelineRunID      uint             `json:"pipelineRunId"`
	CoinSymbol         string           `json:"coinSymbol"`
	Action             string           `json:"action"`
	Confidence         int              `json:"confidence"`
	Leverage           int              `json:"leverage"`
	PositionSizeRatio  decimal.Decimal  `json:"positionSizeRatio"`
	ReferencePrice     *decimal.Decimal `json:"referencePrice"`
	StopLossRatio      *decimal.Decimal `json:"stopLossRatio"`
	StopLossPrice      *decimal.Decimal `json:"stopLossPrice"`
	TakeProfitRatio    *decimal.Decimal `json:"takeProfitRatio"`
	TakeProfitPrice    *decimal.Decimal `json:"takeProfitPrice"`
	Rationale          string           `json:"rationale"`
	ConflictResolution string           `json:"conflictResolution"`
}

// HuntBoardEntryDto is one hunt board row: its id, coin and calculation time, then the verdict itself.
type HuntBoardEntryDto struct {
	ID           uint      `json:"id"`
	CoinSymbol   string    `json:"coinSymbol"`
	CalculatedAt time.Time `json:"calculatedAt"`
	CoinVerdictDto
}
