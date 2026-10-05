package vo

type PipelineRunStatusVo string

const (
	PipelineRunStatusRunning   PipelineRunStatusVo = "running"
	PipelineRunStatusSucceeded PipelineRunStatusVo = "succeeded"
	PipelineRunStatusFailed    PipelineRunStatusVo = "failed"
	// PipelineRunStatusNoData is a clean run that produced nothing to hand downstream.
	PipelineRunStatusNoData PipelineRunStatusVo = "noData"
)
