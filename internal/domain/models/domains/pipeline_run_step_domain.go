package domains

import "github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"

var pipelineRunStepLabels = map[vo.PipelineRunStepVo]string{
	vo.PipelineRunStepDiscovery: "探索",
	vo.PipelineRunStepFiltering: "過濾",
	vo.PipelineRunStepInsight:   "洞察",
	vo.PipelineRunStepVerdict:   "裁決",
}

// PipelineRunStepDomain is a pipeline step as the trader names it.
type PipelineRunStepDomain struct {
	step vo.PipelineRunStepVo
}

func NewPipelineRunStepDomain(step vo.PipelineRunStepVo) PipelineRunStepDomain {
	return PipelineRunStepDomain{step: step}
}

// Label is the step's name in reasons; an unknown step is named as stored.
func (pipelineRunStepDomain PipelineRunStepDomain) Label() string {
	if label, known := pipelineRunStepLabels[pipelineRunStepDomain.step]; known {
		return label
	}

	return string(pipelineRunStepDomain.step)
}

// NotStartedReason says the step was never started, and why.
func (pipelineRunStepDomain PipelineRunStepDomain) NotStartedReason(cause string) string {
	return pipelineRunStepDomain.Label() + "未開始：" + cause
}

// NotSucceededReason says the step ran but did not succeed, and how it ended.
func (pipelineRunStepDomain PipelineRunStepDomain) NotSucceededReason(status string) string {
	return pipelineRunStepDomain.Label() + "未成功：" + status
}
