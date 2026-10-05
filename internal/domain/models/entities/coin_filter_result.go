package entities

import "github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"

// CoinFilterResult is one candidate's outcome in one filtering run; its rule verdicts are stored with it.
type CoinFilterResult struct {
	ID            uint                      `gorm:"primaryKey"`
	PipelineRunID uint                      `gorm:"not null;uniqueIndex:idx_coin_filter_results_run_symbol"`
	CoinSymbol    string                    `gorm:"size:64;not null;uniqueIndex:idx_coin_filter_results_run_symbol"`
	IsKept        bool                      `gorm:"not null;index"`
	Verdicts      []CoinFilterVerdictRecord `gorm:"serializer:json;not null"`
}

// CoinFilterVerdictRecord is one rule's verdict as stored inside its filter result.
type CoinFilterVerdictRecord struct {
	FilterName string `json:"filterName"`
	Outcome    string `json:"outcome"`
	Reason     string `json:"reason"`
}

func (coinFilterResult CoinFilterResult) ToDto() dto.CoinFilterResultDto {
	verdictDtos := make([]dto.CoinFilterVerdictDto, 0, len(coinFilterResult.Verdicts))
	for _, verdict := range coinFilterResult.Verdicts {
		verdictDtos = append(verdictDtos, dto.CoinFilterVerdictDto{
			FilterName: verdict.FilterName, Outcome: verdict.Outcome, Reason: verdict.Reason,
		})
	}

	return dto.CoinFilterResultDto{
		PipelineRunID: coinFilterResult.PipelineRunID,
		CoinSymbol:    coinFilterResult.CoinSymbol,
		IsKept:        coinFilterResult.IsKept,
		Verdicts:      verdictDtos,
	}
}
