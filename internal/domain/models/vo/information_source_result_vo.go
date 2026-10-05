package vo

import "github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"

// InformationSourceResultVo is what one information source returned in one round: its items, or why it failed.
type InformationSourceResultVo struct {
	SourceName       string
	InformationItems []InformationItemVo
	FailureReason    string
}

func (informationSourceResultVo InformationSourceResultVo) Succeeded() bool {
	return informationSourceResultVo.FailureReason == ""
}

func (informationSourceResultVo InformationSourceResultVo) ToInformationSourceOutcome(
	pipelineRunID uint,
) entities.InformationSourceOutcome {
	return entities.InformationSourceOutcome{
		PipelineRunID:        pipelineRunID,
		SourceName:           informationSourceResultVo.SourceName,
		Succeeded:            informationSourceResultVo.Succeeded(),
		InformationItemCount: len(informationSourceResultVo.InformationItems),
		FailureReason:        informationSourceResultVo.FailureReason,
	}
}
