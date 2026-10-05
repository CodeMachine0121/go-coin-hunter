package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
)

//go:generate go tool mockgen -source=i_coin_verdict_repository.go -destination=mocks/mock_i_coin_verdict_repository.go -package=mocks

type ICoinVerdictRepository interface {
	CreateAll(executionContext context.Context, coinVerdicts []entities.CoinVerdict) error
	// FindByPipelineRunID lists a run's verdicts by coin symbol.
	FindByPipelineRunID(executionContext context.Context, pipelineRunID uint) ([]entities.CoinVerdict, error)
}
