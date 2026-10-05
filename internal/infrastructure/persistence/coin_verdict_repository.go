package persistence

import (
	"context"
	"fmt"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
)

type CoinVerdictRepository struct {
	database *gorm.DB
}

func NewCoinVerdictRepository(database *gorm.DB) *CoinVerdictRepository {
	return &CoinVerdictRepository{database: database}
}

func (coinVerdictRepository *CoinVerdictRepository) CreateAll(executionContext context.Context, coinVerdicts []entities.CoinVerdict) error {
	if len(coinVerdicts) == 0 {
		return nil
	}
	if createError := coinVerdictRepository.database.WithContext(executionContext).Create(&coinVerdicts).Error; createError != nil {
		return fmt.Errorf("create coin verdicts: %w", createError)
	}

	return nil
}

func (coinVerdictRepository *CoinVerdictRepository) FindByPipelineRunID(
	executionContext context.Context, pipelineRunID uint,
) ([]entities.CoinVerdict, error) {
	coinVerdicts := []entities.CoinVerdict{}
	if findError := coinVerdictRepository.database.WithContext(executionContext).
		Where(&entities.CoinVerdict{PipelineRunID: pipelineRunID}).Order("coin_symbol").
		Find(&coinVerdicts).Error; findError != nil {
		return nil, fmt.Errorf("find coin verdicts of pipeline run: %w", findError)
	}

	return coinVerdicts, nil
}
