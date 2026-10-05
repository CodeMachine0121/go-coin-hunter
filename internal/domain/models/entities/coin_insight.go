package entities

import "github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"

// CoinInsight is the analyst's normalized judgement on one candidate in one insight run, or why there is none.
type CoinInsight struct {
	ID                      uint   `gorm:"primaryKey"`
	PipelineRunID           uint   `gorm:"not null;uniqueIndex:idx_coin_insights_run_symbol"`
	CoinSymbol              string `gorm:"size:64;not null;uniqueIndex:idx_coin_insights_run_symbol"`
	Succeeded               bool   `gorm:"not null"`
	FailureReason           string
	Direction               string `gorm:"size:16"`
	Strength                int
	Catalyst                string
	Risks                   []string `gorm:"serializer:json;not null"`
	Evidence                []string `gorm:"serializer:json;not null"`
	DataGaps                []string `gorm:"serializer:json;not null"`
	MarketStructureExchange string   `gorm:"size:32"`
}

func (coinInsight CoinInsight) ToDto() dto.CoinInsightDto {
	return dto.CoinInsightDto{
		PipelineRunID:           coinInsight.PipelineRunID,
		CoinSymbol:              coinInsight.CoinSymbol,
		Succeeded:               coinInsight.Succeeded,
		FailureReason:           coinInsight.FailureReason,
		Direction:               coinInsight.Direction,
		Strength:                coinInsight.Strength,
		Catalyst:                coinInsight.Catalyst,
		Risks:                   coinInsight.Risks,
		Evidence:                coinInsight.Evidence,
		DataGaps:                coinInsight.DataGaps,
		MarketStructureExchange: coinInsight.MarketStructureExchange,
	}
}
