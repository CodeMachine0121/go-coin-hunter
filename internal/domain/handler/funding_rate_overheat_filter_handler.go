package handler

import (
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// FundingRateOverheatFilterHandler keeps coins whose longs are not already crowded; a negative rate never rejects.
type FundingRateOverheatFilterHandler struct {
	maximumFundingRate decimal.Decimal
}

func NewFundingRateOverheatFilterHandler(maximumFundingRate decimal.Decimal) *FundingRateOverheatFilterHandler {
	return &FundingRateOverheatFilterHandler{maximumFundingRate: maximumFundingRate}
}

func (fundingRateOverheatFilterHandler *FundingRateOverheatFilterHandler) FilterName() string {
	return "fundingRateOverheat"
}

func (fundingRateOverheatFilterHandler *FundingRateOverheatFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	verdict := vo.FilterVerdictVo{FilterName: fundingRateOverheatFilterHandler.FilterName(), Outcome: vo.FilterOutcomeNoData}
	if coinProfile.MarketStructure == nil {
		verdict.Reason = domains.MarketStructureUnknownReason
		return verdict
	}
	if coinProfile.MarketStructure.FundingRate == nil {
		verdict.Reason = "查不到資金費率"
		return verdict
	}

	fundingRate := *coinProfile.MarketStructure.FundingRate
	verdict.Outcome = vo.FilterOutcomePassed
	if fundingRate.GreaterThan(fundingRateOverheatFilterHandler.maximumFundingRate) {
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = "資金費率 " + domains.NewNumberDescriptionDomain(fundingRate).AsPercentage() +
			" 高於上限 " + domains.NewNumberDescriptionDomain(fundingRateOverheatFilterHandler.maximumFundingRate).AsPercentage()
	}

	return verdict
}
