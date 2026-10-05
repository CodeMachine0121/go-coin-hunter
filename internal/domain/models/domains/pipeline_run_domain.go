package domains

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

const (
	AllInformationSourcesFailedReason = "所有資訊來源皆失敗"
	InterruptedByRestartReason        = "被重啟中斷"
	InformationSourceTimedOutReason   = "連線逾時"
)

// PipelineRunDomain owns a run's status transitions; each method returns the whole record so a transition is never half-applied.
type PipelineRunDomain struct {
	pipelineRun entities.PipelineRun
}

func NewPipelineRunDomain(pipelineRun entities.PipelineRun) PipelineRunDomain {
	return PipelineRunDomain{pipelineRun: pipelineRun}
}

// ConcludeDiscovery: any source succeeding with candidates is success, without candidates is no data, and every source failing is failure.
func (pipelineRunDomain PipelineRunDomain) ConcludeDiscovery(
	informationSourceResults InformationSourceResultsDomain, coinCandidateCount int, finishedAt time.Time,
) entities.PipelineRun {
	if !informationSourceResults.AnySourceSucceeded() {
		return pipelineRunDomain.Fail(AllInformationSourcesFailedReason, finishedAt)
	}

	pipelineRun := pipelineRunDomain.pipelineRun
	pipelineRun.Status = string(vo.PipelineRunStatusSucceeded)
	if coinCandidateCount == 0 {
		pipelineRun.Status = string(vo.PipelineRunStatusNoData)
	}
	pipelineRun.FinishedAt = &finishedAt

	return pipelineRun
}

func (pipelineRunDomain PipelineRunDomain) Fail(failureReason string, finishedAt time.Time) entities.PipelineRun {
	pipelineRun := pipelineRunDomain.pipelineRun
	pipelineRun.Status = string(vo.PipelineRunStatusFailed)
	pipelineRun.FailureReason = failureReason
	pipelineRun.FinishedAt = &finishedAt

	return pipelineRun
}
