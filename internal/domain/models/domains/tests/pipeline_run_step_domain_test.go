package domains_test

import (
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestPipelineRunStepLabels(t *testing.T) {
	assert.Equal(t, []string{"探索", "過濾", "洞察", "裁決", "backtest"}, []string{
		domains.NewPipelineRunStepDomain(vo.PipelineRunStepDiscovery).Label(),
		domains.NewPipelineRunStepDomain(vo.PipelineRunStepFiltering).Label(),
		domains.NewPipelineRunStepDomain(vo.PipelineRunStepInsight).Label(),
		domains.NewPipelineRunStepDomain(vo.PipelineRunStepVerdict).Label(),
		domains.NewPipelineRunStepDomain("backtest").Label(),
	})
	assert.Equal(t, "過濾未成功：noData", domains.NewPipelineRunStepDomain(vo.PipelineRunStepFiltering).NotSucceededReason("noData"))
	assert.Equal(t, "裁決未開始：服務關閉中", domains.NewPipelineRunStepDomain(vo.PipelineRunStepVerdict).NotStartedReason("服務關閉中"))
}
