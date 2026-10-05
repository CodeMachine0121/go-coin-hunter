package service

import (
	"context"
	"fmt"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
)

// PipelineRunService answers about runs across every pipeline step.
type PipelineRunService struct {
	pipelineRunRepository domaininterface.IPipelineRunRepository
	clockProxy            domaininterface.IClockProxy
}

func NewPipelineRunService(
	pipelineRunRepository domaininterface.IPipelineRunRepository, clockProxy domaininterface.IClockProxy,
) *PipelineRunService {
	return &PipelineRunService{pipelineRunRepository: pipelineRunRepository, clockProxy: clockProxy}
}

func (pipelineRunService *PipelineRunService) GetPipelineRuns(executionContext context.Context) ([]dto.PipelineRunDto, error) {
	pipelineRuns, findError := pipelineRunService.pipelineRunRepository.FindAll(executionContext)
	if findError != nil {
		return nil, fmt.Errorf("find pipeline runs: %w", findError)
	}

	pipelineRunDtos := make([]dto.PipelineRunDto, 0, len(pipelineRuns))
	for _, pipelineRun := range pipelineRuns {
		pipelineRunDtos = append(pipelineRunDtos, pipelineRun.ToDto())
	}

	return pipelineRunDtos, nil
}

// FailInterruptedPipelineRuns marks every run still running as failed; called at startup, when nothing can really be running yet.
func (pipelineRunService *PipelineRunService) FailInterruptedPipelineRuns(executionContext context.Context) (int, error) {
	runningPipelineRuns, findError := pipelineRunService.pipelineRunRepository.FindRunning(executionContext)
	if findError != nil {
		return 0, fmt.Errorf("find running pipeline runs: %w", findError)
	}

	finishedAt := pipelineRunService.clockProxy.Now()
	for _, runningPipelineRun := range runningPipelineRuns {
		failedPipelineRun := domains.NewPipelineRunDomain(runningPipelineRun).Fail(domains.InterruptedByRestartReason, finishedAt)
		if updateError := pipelineRunService.pipelineRunRepository.Update(executionContext, failedPipelineRun); updateError != nil {
			return 0, fmt.Errorf("fail interrupted pipeline run %d: %w", runningPipelineRun.ID, updateError)
		}
	}

	return len(runningPipelineRuns), nil
}
