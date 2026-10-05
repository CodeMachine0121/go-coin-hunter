package domains

import (
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// InformationSourceResultsDomain is everything the information sources answered in one discovery round.
type InformationSourceResultsDomain struct {
	informationSourceResults []vo.InformationSourceResultVo
}

func NewInformationSourceResultsDomain(informationSourceResults []vo.InformationSourceResultVo) InformationSourceResultsDomain {
	return InformationSourceResultsDomain{informationSourceResults: informationSourceResults}
}

func (informationSourceResultsDomain InformationSourceResultsDomain) AnySourceSucceeded() bool {
	for _, informationSourceResult := range informationSourceResultsDomain.informationSourceResults {
		if informationSourceResult.FailureReason == "" {
			return true
		}
	}

	return false
}

// InformationSourceOutcomes is one outcome per source, in the order the sources were asked.
func (informationSourceResultsDomain InformationSourceResultsDomain) InformationSourceOutcomes(
	pipelineRunID uint,
) []entities.InformationSourceOutcome {
	informationSourceOutcomes := make([]entities.InformationSourceOutcome, 0, len(informationSourceResultsDomain.informationSourceResults))
	for _, informationSourceResult := range informationSourceResultsDomain.informationSourceResults {
		informationSourceOutcomes = append(informationSourceOutcomes, entities.InformationSourceOutcome{
			PipelineRunID:        pipelineRunID,
			SourceName:           informationSourceResult.SourceName,
			Succeeded:            informationSourceResult.FailureReason == "",
			InformationItemCount: len(informationSourceResult.InformationItems),
			FailureReason:        informationSourceResult.FailureReason,
		})
	}

	return informationSourceOutcomes
}

// CoinIntelligences is every item every source returned, read into the intelligence the system keeps.
func (informationSourceResultsDomain InformationSourceResultsDomain) CoinIntelligences(
	pipelineRunID uint, receivedAt time.Time,
) []entities.CoinIntelligence {
	coinIntelligences := []entities.CoinIntelligence{}
	for _, informationSourceResult := range informationSourceResultsDomain.informationSourceResults {
		for _, informationItem := range informationSourceResult.InformationItems {
			coinIntelligences = append(coinIntelligences,
				NewInformationItemDomain(informationItem).CoinIntelligences(pipelineRunID, receivedAt)...)
		}
	}

	return coinIntelligences
}
