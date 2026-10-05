package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
)

//go:generate go tool mockgen -source=i_coin_filter_result_repository.go -destination=mocks/mock_i_coin_filter_result_repository.go -package=mocks

type ICoinFilterResultRepository interface {
	CreateAll(executionContext context.Context, coinFilterResults []entities.CoinFilterResult) error
	// FindByPipelineRunID lists a run's results by coin symbol.
	FindByPipelineRunID(executionContext context.Context, pipelineRunID uint) ([]entities.CoinFilterResult, error)
}
