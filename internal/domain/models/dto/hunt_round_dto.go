package dto

// HuntRoundDto is one hunt round: every step that ran, in order, and where and why it stopped if it did not finish.
type HuntRoundDto struct {
	Steps         []PipelineRunDto `json:"steps"`
	Completed     bool             `json:"completed"`
	StoppedStep   string           `json:"stoppedStep"`
	StoppedReason string           `json:"stoppedReason"`
}
