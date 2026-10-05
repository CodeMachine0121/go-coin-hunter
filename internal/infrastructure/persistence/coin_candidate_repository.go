package persistence

import (
	"context"
	"fmt"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
)

type CoinCandidateRepository struct {
	database *gorm.DB
}

func NewCoinCandidateRepository(database *gorm.DB) *CoinCandidateRepository {
	return &CoinCandidateRepository{database: database}
}

func (coinCandidateRepository *CoinCandidateRepository) CreateAll(
	executionContext context.Context, coinCandidates []entities.CoinCandidate,
) error {
	if len(coinCandidates) == 0 {
		return nil
	}
	if createError := coinCandidateRepository.database.WithContext(executionContext).
		Create(&coinCandidates).Error; createError != nil {
		return fmt.Errorf("create coin candidates: %w", createError)
	}

	return nil
}

func (coinCandidateRepository *CoinCandidateRepository) FindByPipelineRunID(
	executionContext context.Context, pipelineRunID uint,
) ([]entities.CoinCandidate, error) {
	coinCandidates := []entities.CoinCandidate{}
	if findError := coinCandidateRepository.database.WithContext(executionContext).
		Where(&entities.CoinCandidate{PipelineRunID: pipelineRunID}).Order("coin_symbol").
		Find(&coinCandidates).Error; findError != nil {
		return nil, fmt.Errorf("find coin candidates of pipeline run: %w", findError)
	}

	return coinCandidates, nil
}
