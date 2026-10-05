package application

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
)

type CoinInsightApplication struct {
	coinInsightService *service.CoinInsightService
}

func NewCoinInsightApplication(coinInsightService *service.CoinInsightService) *CoinInsightApplication {
	return &CoinInsightApplication{coinInsightService: coinInsightService}
}

// AnalyzeCoinCandidatesManually is the trader asking for an insight round right now.
func (coinInsightApplication *CoinInsightApplication) AnalyzeCoinCandidatesManually(executionContext context.Context) (dto.PipelineRunDto, error) {
	return coinInsightApplication.coinInsightService.AnalyzeCoinCandidates(executionContext, vo.PipelineRunTriggerSourceManual)
}

func (coinInsightApplication *CoinInsightApplication) GetLatestCoinInsights(executionContext context.Context) ([]dto.CoinInsightDto, error) {
	return coinInsightApplication.coinInsightService.GetLatestCoinInsights(executionContext)
}

func (coinInsightApplication *CoinInsightApplication) GetCoinInsightsOfPipelineRun(
	executionContext context.Context, pipelineRunID uint,
) ([]dto.CoinInsightDto, error) {
	return coinInsightApplication.coinInsightService.GetCoinInsightsOfPipelineRun(executionContext, pipelineRunID)
}
