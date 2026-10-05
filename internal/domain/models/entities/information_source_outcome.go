package entities

import "github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"

// InformationSourceOutcome records how one information source fared in one discovery run.
type InformationSourceOutcome struct {
	ID                   uint   `gorm:"primaryKey"`
	PipelineRunID        uint   `gorm:"not null;index"`
	SourceName           string `gorm:"size:64;not null"`
	Succeeded            bool   `gorm:"not null"`
	InformationItemCount int    `gorm:"not null"`
	FailureReason        string
}

func (informationSourceOutcome InformationSourceOutcome) ToDto() dto.InformationSourceOutcomeDto {
	return dto.InformationSourceOutcomeDto{
		SourceName:           informationSourceOutcome.SourceName,
		Succeeded:            informationSourceOutcome.Succeeded,
		InformationItemCount: informationSourceOutcome.InformationItemCount,
		FailureReason:        informationSourceOutcome.FailureReason,
	}
}
