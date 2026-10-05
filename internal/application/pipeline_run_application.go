package application

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
)

type PipelineRunApplication struct {
	pipelineRunService *service.PipelineRunService
}

func NewPipelineRunApplication(pipelineRunService *service.PipelineRunService) *PipelineRunApplication {
	return &PipelineRunApplication{pipelineRunService: pipelineRunService}
}

func (pipelineRunApplication *PipelineRunApplication) GetPipelineRuns(executionContext context.Context) ([]dto.PipelineRunDto, error) {
	return pipelineRunApplication.pipelineRunService.GetPipelineRuns(executionContext)
}

// FailInterruptedPipelineRuns returns how many runs a restart cut short.
func (pipelineRunApplication *PipelineRunApplication) FailInterruptedPipelineRuns(executionContext context.Context) (int, error) {
	return pipelineRunApplication.pipelineRunService.FailInterruptedPipelineRuns(executionContext)
}
