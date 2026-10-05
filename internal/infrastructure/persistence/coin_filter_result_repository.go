package persistence

import (
	"context"
	"fmt"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
)

type CoinFilterResultRepository struct {
	database *gorm.DB
}

func NewCoinFilterResultRepository(database *gorm.DB) *CoinFilterResultRepository {
	return &CoinFilterResultRepository{database: database}
}

func (coinFilterResultRepository *CoinFilterResultRepository) CreateAll(
	executionContext context.Context, coinFilterResults []entities.CoinFilterResult,
) error {
	if len(coinFilterResults) == 0 {
		return nil
	}
	if createError := coinFilterResultRepository.database.WithContext(executionContext).
		Create(&coinFilterResults).Error; createError != nil {
		return fmt.Errorf("create coin filter results: %w", createError)
	}

	return nil
}

func (coinFilterResultRepository *CoinFilterResultRepository) FindByPipelineRunID(
	executionContext context.Context, pipelineRunID uint,
) ([]entities.CoinFilterResult, error) {
	coinFilterResults := []entities.CoinFilterResult{}
	if findError := coinFilterResultRepository.database.WithContext(executionContext).
		Where(&entities.CoinFilterResult{PipelineRunID: pipelineRunID}).Order("coin_symbol").
		Find(&coinFilterResults).Error; findError != nil {
		return nil, fmt.Errorf("find coin filter results of pipeline run: %w", findError)
	}

	return coinFilterResults, nil
}
