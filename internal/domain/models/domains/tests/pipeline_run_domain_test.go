package domains_test

import (
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestConcludeDiscovery(t *testing.T) {
	succeeded := vo.InformationSourceResultVo{SourceName: "a"}
	failed := vo.InformationSourceResultVo{SourceName: "b", FailureReason: "連線逾時"}
	testCases := []struct {
		name              string
		results           []vo.InformationSourceResultVo
		candidateCount    int
		wantStatus        vo.PipelineRunStatusVo
		wantFailureReason string
	}{
		{name: "a source succeeding with candidates", results: []vo.InformationSourceResultVo{succeeded, failed}, candidateCount: 1, wantStatus: vo.PipelineRunStatusSucceeded},
		{name: "a source succeeding without candidates", results: []vo.InformationSourceResultVo{succeeded}, candidateCount: 0, wantStatus: vo.PipelineRunStatusNoData},
		{name: "every source failing", results: []vo.InformationSourceResultVo{failed}, candidateCount: 0,
			wantStatus: vo.PipelineRunStatusFailed, wantFailureReason: domains.AllInformationSourcesFailedReason},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pipelineRun := domains.NewPipelineRunDomain(entities.PipelineRun{ID: 1, Status: string(vo.PipelineRunStatusRunning)}).
				ConcludeDiscovery(domains.NewInformationSourceResultsDomain(testCase.results), testCase.candidateCount, receivedAt)

			assert.Equal(t, string(testCase.wantStatus), pipelineRun.Status)
			assert.Equal(t, testCase.wantFailureReason, pipelineRun.FailureReason)
			assert.Equal(t, receivedAt, *pipelineRun.FinishedAt)
		})
	}
}
