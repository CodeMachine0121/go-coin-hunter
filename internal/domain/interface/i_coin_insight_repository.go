package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
)

//go:generate go tool mockgen -source=i_coin_insight_repository.go -destination=mocks/mock_i_coin_insight_repository.go -package=mocks

type ICoinInsightRepository interface {
	CreateAll(executionContext context.Context, coinInsights []entities.CoinInsight) error
	// FindByPipelineRunID lists a run's insights by coin symbol.
	FindByPipelineRunID(executionContext context.Context, pipelineRunID uint) ([]entities.CoinInsight, error)
}
