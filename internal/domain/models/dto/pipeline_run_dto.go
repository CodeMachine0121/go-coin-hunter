package dto

import "time"

type PipelineRunDto struct {
	ID                        uint                          `json:"id"`
	Step                      string                        `json:"step"`
	TriggerSource             string                        `json:"triggerSource"`
	Status                    string                        `json:"status"`
	FailureReason             string                        `json:"failureReason"`
	TriggeredByPipelineRunID  *uint                         `json:"triggeredByPipelineRunId"`
	StartedAt                 time.Time                     `json:"startedAt"`
	FinishedAt                *time.Time                    `json:"finishedAt"`
	InformationSourceOutcomes []InformationSourceOutcomeDto `json:"informationSourceOutcomes"`
}
