package application

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
)

type CoinDiscoveryApplication struct {
	coinDiscoveryService *service.CoinDiscoveryService
}

func NewCoinDiscoveryApplication(coinDiscoveryService *service.CoinDiscoveryService) *CoinDiscoveryApplication {
	return &CoinDiscoveryApplication{coinDiscoveryService: coinDiscoveryService}
}

// DiscoverCoinsManually is the trader asking for a discovery round right now.
func (coinDiscoveryApplication *CoinDiscoveryApplication) DiscoverCoinsManually(
	executionContext context.Context,
) (dto.PipelineRunDto, error) {
	return coinDiscoveryApplication.coinDiscoveryService.DiscoverCoins(executionContext, vo.PipelineRunTriggerSourceManual)
}

func (coinDiscoveryApplication *CoinDiscoveryApplication) GetLatestCoinCandidates(
	executionContext context.Context,
) ([]dto.CoinCandidateDto, error) {
	return coinDiscoveryApplication.coinDiscoveryService.GetLatestCoinCandidates(executionContext)
}

func (coinDiscoveryApplication *CoinDiscoveryApplication) GetCoinIntelligencesOfPipelineRun(
	executionContext context.Context, pipelineRunID uint,
) ([]dto.CoinIntelligenceDto, error) {
	return coinDiscoveryApplication.coinDiscoveryService.GetCoinIntelligencesOfPipelineRun(executionContext, pipelineRunID)
}
