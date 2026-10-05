package _interface

import (
	"context"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_coin_intelligence_repository.go -destination=mocks/mock_i_coin_intelligence_repository.go -package=mocks

type ICoinIntelligenceRepository interface {
	// SaveNew keeps only intelligence not stored before (same source, message and coin); the rest are skipped silently.
	SaveNew(executionContext context.Context, coinIntelligences []entities.CoinIntelligence) error
	// FindPublishedSince includes intelligence published exactly at the given time.
	FindPublishedSince(executionContext context.Context, publishedSince time.Time) ([]entities.CoinIntelligence, error)
	FindByPipelineRunID(executionContext context.Context, pipelineRunID uint) ([]entities.CoinIntelligence, error)
	// FindDeclaredContractAddresses returns, per coin symbol, the chain and address most recently declared by an on-chain source.
	FindDeclaredContractAddresses(executionContext context.Context, coinSymbols []string) (map[string]vo.TokenAddressVo, error)
}
