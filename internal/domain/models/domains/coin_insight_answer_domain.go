package domains

import (
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

const (
	minimumInsightStrength = 1
	maximumInsightStrength = 10
)

// CoinInsightAnswerDomain turns an analyst's answer into an insight that can be trusted downstream.
type CoinInsightAnswerDomain struct {
	answer   vo.CoinInsightAnswerVo
	material vo.CoinInsightMaterialVo
}

func NewCoinInsightAnswerDomain(answer vo.CoinInsightAnswerVo, material vo.CoinInsightMaterialVo) CoinInsightAnswerDomain {
	return CoinInsightAnswerDomain{answer: answer, material: material}
}

// ToCoinInsight keeps only known directions (anything else is neutral), clamps strength into 1-10, and lists the
// gathering gaps before the analyst's own, each once.
func (coinInsightAnswerDomain CoinInsightAnswerDomain) ToCoinInsight(pipelineRunID uint) entities.CoinInsight {
	answer := coinInsightAnswerDomain.answer
	direction := vo.CoinInsightDirectionVo(strings.ToLower(strings.TrimSpace(answer.Direction)))
	if direction != vo.CoinInsightDirectionBullish && direction != vo.CoinInsightDirectionBearish {
		direction = vo.CoinInsightDirectionNeutral
	}

	dataGaps := []string{}
	seenDataGaps := map[string]bool{}
	for _, dataGap := range append(append([]string{}, coinInsightAnswerDomain.material.DataGaps...), answer.DataGaps...) {
		trimmedDataGap := strings.TrimSpace(dataGap)
		if trimmedDataGap == "" || seenDataGaps[trimmedDataGap] {
			continue
		}
		seenDataGaps[trimmedDataGap] = true
		dataGaps = append(dataGaps, trimmedDataGap)
	}

	coinInsight := entities.CoinInsight{
		PipelineRunID: pipelineRunID,
		CoinSymbol:    coinInsightAnswerDomain.material.CoinSymbol,
		Succeeded:     true,
		Direction:     string(direction),
		Strength:      min(max(answer.Strength, minimumInsightStrength), maximumInsightStrength),
		Catalyst:      strings.TrimSpace(answer.Catalyst),
		Risks:         append([]string{}, answer.Risks...),
		Evidence:      append([]string{}, answer.Evidence...),
		DataGaps:      dataGaps,
	}
	if coinInsightAnswerDomain.material.MarketStructure != nil {
		coinInsight.MarketStructureExchange = coinInsightAnswerDomain.material.MarketStructure.ExchangeName
	}

	return coinInsight
}
