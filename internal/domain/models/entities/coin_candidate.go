package entities

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
)

// CoinCandidate is one new coin a discovery run surfaced; one per coin symbol per run.
type CoinCandidate struct {
	ID                  uint      `gorm:"primaryKey"`
	PipelineRunID       uint      `gorm:"not null;uniqueIndex:idx_coin_candidates_run_symbol"`
	CoinSymbol          string    `gorm:"size:64;not null;uniqueIndex:idx_coin_candidates_run_symbol"`
	SourceCount         int       `gorm:"not null"`
	IntelligenceCount   int       `gorm:"not null"`
	EarliestMentionedAt time.Time `gorm:"not null"`
}

func (coinCandidate CoinCandidate) ToDto() dto.CoinCandidateDto {
	return dto.CoinCandidateDto{
		PipelineRunID:       coinCandidate.PipelineRunID,
		CoinSymbol:          coinCandidate.CoinSymbol,
		SourceCount:         coinCandidate.SourceCount,
		IntelligenceCount:   coinCandidate.IntelligenceCount,
		EarliestMentionedAt: coinCandidate.EarliestMentionedAt,
	}
}
