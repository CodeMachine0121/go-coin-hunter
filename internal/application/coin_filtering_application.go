package application

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
)

type CoinFilteringApplication struct {
	coinFilteringService *service.CoinFilteringService
}

func NewCoinFilteringApplication(coinFilteringService *service.CoinFilteringService) *CoinFilteringApplication {
	return &CoinFilteringApplication{coinFilteringService: coinFilteringService}
}

// FilterCoinCandidatesManually is the trader asking for a filtering round right now.
func (coinFilteringApplication *CoinFilteringApplication) FilterCoinCandidatesManually(
	executionContext context.Context,
) (dto.PipelineRunDto, error) {
	return coinFilteringApplication.coinFilteringService.FilterCoinCandidates(executionContext, vo.PipelineRunTriggerSourceManual)
}

func (coinFilteringApplication *CoinFilteringApplication) GetLatestKeptCoinCandidates(
	executionContext context.Context,
) ([]dto.CoinFilterResultDto, error) {
	return coinFilteringApplication.coinFilteringService.GetLatestKeptCoinCandidates(executionContext)
}

func (coinFilteringApplication *CoinFilteringApplication) GetCoinFilterResultsOfPipelineRun(
	executionContext context.Context, pipelineRunID uint,
) ([]dto.CoinFilterResultDto, error) {
	return coinFilteringApplication.coinFilteringService.GetCoinFilterResultsOfPipelineRun(executionContext, pipelineRunID)
}
