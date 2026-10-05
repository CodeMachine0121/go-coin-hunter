package vo

// PipelineRunStepVo names which step of the hunt pipeline a run belongs to.
type PipelineRunStepVo string

const (
	PipelineRunStepDiscovery PipelineRunStepVo = "discovery"
	PipelineRunStepFiltering PipelineRunStepVo = "filtering"
	PipelineRunStepInsight   PipelineRunStepVo = "insight"
)
