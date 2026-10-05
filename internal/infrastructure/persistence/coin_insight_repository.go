package persistence

import (
	"context"
	"fmt"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
)

type CoinInsightRepository struct {
	database *gorm.DB
}

func NewCoinInsightRepository(database *gorm.DB) *CoinInsightRepository {
	return &CoinInsightRepository{database: database}
}

func (coinInsightRepository *CoinInsightRepository) CreateAll(executionContext context.Context, coinInsights []entities.CoinInsight) error {
	if len(coinInsights) == 0 {
		return nil
	}
	if createError := coinInsightRepository.database.WithContext(executionContext).Create(&coinInsights).Error; createError != nil {
		return fmt.Errorf("create coin insights: %w", createError)
	}

	return nil
}

func (coinInsightRepository *CoinInsightRepository) FindByPipelineRunID(
	executionContext context.Context, pipelineRunID uint,
) ([]entities.CoinInsight, error) {
	coinInsights := []entities.CoinInsight{}
	if findError := coinInsightRepository.database.WithContext(executionContext).
		Where(&entities.CoinInsight{PipelineRunID: pipelineRunID}).Order("coin_symbol").
		Find(&coinInsights).Error; findError != nil {
		return nil, fmt.Errorf("find coin insights of pipeline run: %w", findError)
	}

	return coinInsights, nil
}
