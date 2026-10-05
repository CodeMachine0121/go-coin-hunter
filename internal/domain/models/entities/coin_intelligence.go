package entities

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
)

// CoinIntelligence is one stored message about at most one coin; the same message about the same coin from the same source is kept once.
type CoinIntelligence struct {
	ID                 uint   `gorm:"primaryKey"`
	PipelineRunID      uint   `gorm:"not null;index"`
	SourceName         string `gorm:"size:64;not null;uniqueIndex:idx_coin_intelligences_message"`
	ExternalIdentifier string `gorm:"size:512;not null;uniqueIndex:idx_coin_intelligences_message"`
	// CoinSymbol is empty when the message names no recognizable coin.
	CoinSymbol         string `gorm:"size:64;not null;uniqueIndex:idx_coin_intelligences_message;index"`
	Title              string
	Link               string
	PublishedAt        time.Time `gorm:"not null;index"`
	IsTraditionalAsset bool      `gorm:"not null"`
	// ChainID and ContractAddress are empty unless an on-chain source declared them; the address keeps its original case.
	ChainID         string `gorm:"size:64"`
	ContractAddress string `gorm:"size:128"`
}

func (coinIntelligence CoinIntelligence) ToDto() dto.CoinIntelligenceDto {
	return dto.CoinIntelligenceDto{
		SourceName:         coinIntelligence.SourceName,
		CoinSymbol:         coinIntelligence.CoinSymbol,
		Title:              coinIntelligence.Title,
		Link:               coinIntelligence.Link,
		PublishedAt:        coinIntelligence.PublishedAt,
		IsTraditionalAsset: coinIntelligence.IsTraditionalAsset,
	}
}
