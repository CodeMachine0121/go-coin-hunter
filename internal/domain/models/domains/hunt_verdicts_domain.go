package domains

import (
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

const (
	missingVerdictRationale  = "CIO 未給出裁決"
	missingPriceRationale    = "查不到最新價格，無法設定停損"
	minimumVerdictConfidence = 0
	maximumVerdictConfidence = 100
)

var percentPerUnit = decimal.NewFromInt(100)

// HuntVerdictsDomain turns the strategist's answers into one trustworthy verdict per coin it was shown.
type HuntVerdictsDomain struct {
	materials     []vo.HuntVerdictMaterialVo
	answers       []vo.HuntVerdictAnswerVo
	verdictPolicy vo.HuntVerdictPolicyVo
}

func NewHuntVerdictsDomain(
	materials []vo.HuntVerdictMaterialVo, answers []vo.HuntVerdictAnswerVo, verdictPolicy vo.HuntVerdictPolicyVo,
) HuntVerdictsDomain {
	return HuntVerdictsDomain{materials: materials, answers: answers, verdictPolicy: verdictPolicy}
}

// ToCoinVerdicts yields exactly one verdict per material coin, in material order: answers are matched without case,
// a coin left unanswered is watched, and answers about coins never shown are dropped.
func (huntVerdictsDomain HuntVerdictsDomain) ToCoinVerdicts(pipelineRunID uint) []entities.CoinVerdict {
	answersBySymbol := map[string]vo.HuntVerdictAnswerVo{}
	for _, answer := range huntVerdictsDomain.answers {
		upperSymbol := strings.ToUpper(strings.TrimSpace(answer.CoinSymbol))
		if _, seen := answersBySymbol[upperSymbol]; !seen {
			answersBySymbol[upperSymbol] = answer
		}
	}

	policy := huntVerdictsDomain.verdictPolicy
	coinVerdicts := make([]entities.CoinVerdict, 0, len(huntVerdictsDomain.materials))
	for _, material := range huntVerdictsDomain.materials {
		coinVerdict := entities.CoinVerdict{PipelineRunID: pipelineRunID, CoinSymbol: material.CoinSymbol,
			Action: string(vo.HuntActionWatch), PositionSizeRatio: decimal.Zero}
		answer, answered := answersBySymbol[strings.ToUpper(material.CoinSymbol)]
		if !answered {
			coinVerdict.Rationale = missingVerdictRationale
			coinVerdicts = append(coinVerdicts, coinVerdict)
			continue
		}

		coinVerdict.Confidence = min(max(answer.Confidence, minimumVerdictConfidence), maximumVerdictConfidence)
		coinVerdict.Rationale = strings.TrimSpace(answer.Rationale)
		coinVerdict.ConflictResolution = strings.TrimSpace(answer.ConflictResolution)
		action := vo.HuntActionVo(strings.ToLower(strings.TrimSpace(answer.Action)))
		if action == vo.HuntActionWatch || action == vo.HuntActionAvoid {
			coinVerdict.Action = string(action)
		}
		if action != vo.HuntActionLong && action != vo.HuntActionShort {
			coinVerdicts = append(coinVerdicts, coinVerdict)
			continue
		}

		var lastPrice *decimal.Decimal
		if material.MarketStructure != nil {
			lastPrice = material.MarketStructure.LastPrice
		}
		if lastPrice == nil || !lastPrice.IsPositive() {
			coinVerdict.Rationale = missingPriceRationale
			coinVerdicts = append(coinVerdicts, coinVerdict)
			continue
		}

		stopLossRatio := decimal.Max(policy.MinimumStopLossPercent, decimal.Min(policy.MaximumStopLossPercent, answer.StopLossPercent)).Div(percentPerUnit)
		takeProfitRatio := decimal.Max(policy.MinimumTakeProfitPercent, decimal.Min(policy.MaximumTakeProfitPercent, answer.TakeProfitPercent)).Div(percentPerUnit)
		stopLossPrice := lastPrice.Mul(decimal.NewFromInt(1).Sub(stopLossRatio))
		takeProfitPrice := lastPrice.Mul(decimal.NewFromInt(1).Add(takeProfitRatio))
		if action == vo.HuntActionShort {
			stopLossPrice = lastPrice.Mul(decimal.NewFromInt(1).Add(stopLossRatio))
			takeProfitPrice = lastPrice.Mul(decimal.NewFromInt(1).Sub(takeProfitRatio))
		}
		referencePrice := *lastPrice
		coinVerdict.Action = string(action)
		coinVerdict.Leverage = min(max(answer.Leverage, policy.MinimumLeverage), policy.MaximumLeverage)
		coinVerdict.PositionSizeRatio = decimal.Max(decimal.Zero, decimal.Min(policy.MaximumPositionSizePercent, answer.PositionSizePercent)).Div(percentPerUnit)
		coinVerdict.ReferencePrice = &referencePrice
		coinVerdict.StopLossRatio = &stopLossRatio
		coinVerdict.StopLossPrice = &stopLossPrice
		coinVerdict.TakeProfitRatio = &takeProfitRatio
		coinVerdict.TakeProfitPrice = &takeProfitPrice
		coinVerdicts = append(coinVerdicts, coinVerdict)
	}

	return coinVerdicts
}
