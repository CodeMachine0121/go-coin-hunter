package entities

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
)

// PipelineRun records one execution of one hunt pipeline step.
type PipelineRun struct {
	ID            uint   `gorm:"primaryKey"`
	Step          string `gorm:"size:32;not null;index"`
	TriggerSource string `gorm:"size:16;not null"`
	Status        string `gorm:"size:16;not null;index"`
	FailureReason string
	// TriggeredByPipelineRunID links a downstream step to the upstream run it consumed.
	TriggeredByPipelineRunID *uint
	StartedAt                time.Time `gorm:"not null;index"`
	FinishedAt               *time.Time

	InformationSourceOutcomes []InformationSourceOutcome `gorm:"foreignKey:PipelineRunID"`
}

func (pipelineRun PipelineRun) ToDto() dto.PipelineRunDto {
	informationSourceOutcomeDtos := make([]dto.InformationSourceOutcomeDto, 0, len(pipelineRun.InformationSourceOutcomes))
	for _, informationSourceOutcome := range pipelineRun.InformationSourceOutcomes {
		informationSourceOutcomeDtos = append(informationSourceOutcomeDtos, informationSourceOutcome.ToDto())
	}

	return dto.PipelineRunDto{
		ID:                        pipelineRun.ID,
		Step:                      pipelineRun.Step,
		TriggerSource:             pipelineRun.TriggerSource,
		Status:                    pipelineRun.Status,
		FailureReason:             pipelineRun.FailureReason,
		TriggeredByPipelineRunID:  pipelineRun.TriggeredByPipelineRunID,
		StartedAt:                 pipelineRun.StartedAt,
		FinishedAt:                pipelineRun.FinishedAt,
		InformationSourceOutcomes: informationSourceOutcomeDtos,
	}
}
