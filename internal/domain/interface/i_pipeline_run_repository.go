package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
)

//go:generate go tool mockgen -source=i_pipeline_run_repository.go -destination=mocks/mock_i_pipeline_run_repository.go -package=mocks

type IPipelineRunRepository interface {
	Create(executionContext context.Context, pipelineRun entities.PipelineRun) (entities.PipelineRun, error)
	Update(executionContext context.Context, pipelineRun entities.PipelineRun) error
	// FindOne returns domains.ErrPipelineRunNotFound when no run has that id.
	FindOne(executionContext context.Context, pipelineRunID uint) (entities.PipelineRun, error)
	// FindLatestSucceeded returns found=false when the step has never succeeded.
	FindLatestSucceeded(executionContext context.Context, step string) (pipelineRun entities.PipelineRun, found bool, findError error)
	// FindAll lists every run newest first, with its information source outcomes.
	FindAll(executionContext context.Context) ([]entities.PipelineRun, error)
	FindRunning(executionContext context.Context) ([]entities.PipelineRun, error)
}
