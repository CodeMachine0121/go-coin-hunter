package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
)

//go:generate go tool mockgen -source=i_coin_candidate_repository.go -destination=mocks/mock_i_coin_candidate_repository.go -package=mocks

type ICoinCandidateRepository interface {
	CreateAll(executionContext context.Context, coinCandidates []entities.CoinCandidate) error
	FindByPipelineRunID(executionContext context.Context, pipelineRunID uint) ([]entities.CoinCandidate, error)
}
