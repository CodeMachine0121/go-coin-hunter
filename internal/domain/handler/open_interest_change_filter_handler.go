package handler

import (
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// OpenInterestChangeFilterHandler keeps coins whose open interest is not draining: money leaving is no time to go long.
type OpenInterestChangeFilterHandler struct {
	minimumOpenInterestChangeRatio decimal.Decimal
}

func NewOpenInterestChangeFilterHandler(minimumOpenInterestChangeRatio decimal.Decimal) *OpenInterestChangeFilterHandler {
	return &OpenInterestChangeFilterHandler{minimumOpenInterestChangeRatio: minimumOpenInterestChangeRatio}
}

func (openInterestChangeFilterHandler *OpenInterestChangeFilterHandler) FilterName() string {
	return "openInterestChange"
}

func (openInterestChangeFilterHandler *OpenInterestChangeFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	verdict := vo.FilterVerdictVo{FilterName: openInterestChangeFilterHandler.FilterName(), Outcome: vo.FilterOutcomeNoData}
	if coinProfile.MarketStructure == nil {
		verdict.Reason = domains.MarketStructureUnknownReason
		return verdict
	}
	if coinProfile.MarketStructure.OpenInterestChangeRatio24h == nil {
		verdict.Reason = "查不到持倉量 24 小時變化"
		return verdict
	}

	openInterestChangeRatio := *coinProfile.MarketStructure.OpenInterestChangeRatio24h
	verdict.Outcome = vo.FilterOutcomePassed
	if openInterestChangeRatio.LessThan(openInterestChangeFilterHandler.minimumOpenInterestChangeRatio) {
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = "持倉量 24 小時減少 " + domains.NewNumberDescriptionDomain(openInterestChangeRatio.Neg()).AsPercentage() +
			" 超過下限 " + domains.NewNumberDescriptionDomain(openInterestChangeFilterHandler.minimumOpenInterestChangeRatio.Neg()).AsPercentage()
	}

	return verdict
}
