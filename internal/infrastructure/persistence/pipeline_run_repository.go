package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"gorm.io/gorm"
)

type PipelineRunRepository struct {
	database *gorm.DB
}

func NewPipelineRunRepository(database *gorm.DB) *PipelineRunRepository {
	return &PipelineRunRepository{database: database}
}

func (pipelineRunRepository *PipelineRunRepository) Create(
	executionContext context.Context, pipelineRun entities.PipelineRun,
) (entities.PipelineRun, error) {
	if createError := pipelineRunRepository.database.WithContext(executionContext).
		Omit("InformationSourceOutcomes").Create(&pipelineRun).Error; createError != nil {
		return entities.PipelineRun{}, fmt.Errorf("create pipeline run: %w", createError)
	}

	return pipelineRun, nil
}

func (pipelineRunRepository *PipelineRunRepository) Update(
	executionContext context.Context, pipelineRun entities.PipelineRun,
) error {
	if updateError := pipelineRunRepository.database.WithContext(executionContext).
		Omit("InformationSourceOutcomes").Save(&pipelineRun).Error; updateError != nil {
		return fmt.Errorf("update pipeline run: %w", updateError)
	}

	return nil
}

func (pipelineRunRepository *PipelineRunRepository) FindOne(
	executionContext context.Context, pipelineRunID uint,
) (entities.PipelineRun, error) {
	pipelineRun := entities.PipelineRun{}
	findError := pipelineRunRepository.database.WithContext(executionContext).
		Preload("InformationSourceOutcomes").First(&pipelineRun, pipelineRunID).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return entities.PipelineRun{}, domains.ErrPipelineRunNotFound
	}
	if findError != nil {
		return entities.PipelineRun{}, fmt.Errorf("find pipeline run: %w", findError)
	}

	return pipelineRun, nil
}

func (pipelineRunRepository *PipelineRunRepository) FindLatestSucceeded(
	executionContext context.Context, step string,
) (entities.PipelineRun, bool, error) {
	pipelineRuns := []entities.PipelineRun{}
	if findError := pipelineRunRepository.database.WithContext(executionContext).
		Where(&entities.PipelineRun{Step: step, Status: string(vo.PipelineRunStatusSucceeded)}).
		Order("started_at DESC").Order("id DESC").Limit(1).Find(&pipelineRuns).Error; findError != nil {
		return entities.PipelineRun{}, false, fmt.Errorf("find latest succeeded pipeline run: %w", findError)
	}
	if len(pipelineRuns) == 0 {
		return entities.PipelineRun{}, false, nil
	}

	return pipelineRuns[0], true, nil
}

func (pipelineRunRepository *PipelineRunRepository) FindAll(executionContext context.Context) ([]entities.PipelineRun, error) {
	pipelineRuns := []entities.PipelineRun{}
	if findError := pipelineRunRepository.database.WithContext(executionContext).
		Preload("InformationSourceOutcomes").
		Order("started_at DESC").Order("id DESC").Find(&pipelineRuns).Error; findError != nil {
		return nil, fmt.Errorf("find pipeline runs: %w", findError)
	}

	return pipelineRuns, nil
}

func (pipelineRunRepository *PipelineRunRepository) FindRunning(executionContext context.Context) ([]entities.PipelineRun, error) {
	pipelineRuns := []entities.PipelineRun{}
	if findError := pipelineRunRepository.database.WithContext(executionContext).
		Where(&entities.PipelineRun{Status: string(vo.PipelineRunStatusRunning)}).
		Find(&pipelineRuns).Error; findError != nil {
		return nil, fmt.Errorf("find running pipeline runs: %w", findError)
	}

	return pipelineRuns, nil
}
