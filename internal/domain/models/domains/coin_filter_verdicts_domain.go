package domains

import (
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// CoinFilterVerdictsDomain is every rule's verdict on one candidate.
type CoinFilterVerdictsDomain struct {
	filterVerdicts []vo.FilterVerdictVo
}

func NewCoinFilterVerdictsDomain(filterVerdicts []vo.FilterVerdictVo) CoinFilterVerdictsDomain {
	return CoinFilterVerdictsDomain{filterVerdicts: filterVerdicts}
}

// IsKept: any rejection drops the candidate; missing data never does.
func (coinFilterVerdictsDomain CoinFilterVerdictsDomain) IsKept() bool {
	for _, filterVerdict := range coinFilterVerdictsDomain.filterVerdicts {
		if filterVerdict.Outcome == vo.FilterOutcomeRejected {
			return false
		}
	}

	return true
}

func (coinFilterVerdictsDomain CoinFilterVerdictsDomain) ToCoinFilterResult(
	pipelineRunID uint, coinSymbol string,
) entities.CoinFilterResult {
	verdictRecords := make([]entities.CoinFilterVerdictRecord, 0, len(coinFilterVerdictsDomain.filterVerdicts))
	for _, filterVerdict := range coinFilterVerdictsDomain.filterVerdicts {
		verdictRecords = append(verdictRecords, entities.CoinFilterVerdictRecord{
			FilterName: filterVerdict.FilterName, Outcome: string(filterVerdict.Outcome), Reason: filterVerdict.Reason,
		})
	}

	return entities.CoinFilterResult{
		PipelineRunID: pipelineRunID,
		CoinSymbol:    coinSymbol,
		IsKept:        coinFilterVerdictsDomain.IsKept(),
		Verdicts:      verdictRecords,
	}
}
