package persistence

import (
	"context"
	"fmt"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
)

type InformationSourceOutcomeRepository struct {
	database *gorm.DB
}

func NewInformationSourceOutcomeRepository(database *gorm.DB) *InformationSourceOutcomeRepository {
	return &InformationSourceOutcomeRepository{database: database}
}

func (informationSourceOutcomeRepository *InformationSourceOutcomeRepository) CreateAll(
	executionContext context.Context, informationSourceOutcomes []entities.InformationSourceOutcome,
) error {
	if len(informationSourceOutcomes) == 0 {
		return nil
	}
	if createError := informationSourceOutcomeRepository.database.WithContext(executionContext).
		Create(&informationSourceOutcomes).Error; createError != nil {
		return fmt.Errorf("create information source outcomes: %w", createError)
	}

	return nil
}
