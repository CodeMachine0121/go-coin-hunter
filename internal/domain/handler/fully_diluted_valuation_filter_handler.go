package handler

import (
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// FullyDilutedValuationFilterHandler keeps coins big enough not to be trivially manipulated and small enough to still have room to run.
type FullyDilutedValuationFilterHandler struct {
	minimumFullyDilutedValuationUsd decimal.Decimal
	maximumFullyDilutedValuationUsd decimal.Decimal
}

func NewFullyDilutedValuationFilterHandler(
	minimumFullyDilutedValuationUsd decimal.Decimal, maximumFullyDilutedValuationUsd decimal.Decimal,
) *FullyDilutedValuationFilterHandler {
	return &FullyDilutedValuationFilterHandler{
		minimumFullyDilutedValuationUsd: minimumFullyDilutedValuationUsd,
		maximumFullyDilutedValuationUsd: maximumFullyDilutedValuationUsd,
	}
}

func (fullyDilutedValuationFilterHandler *FullyDilutedValuationFilterHandler) FilterName() string {
	return "fullyDilutedValuation"
}

func (fullyDilutedValuationFilterHandler *FullyDilutedValuationFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	verdict := vo.FilterVerdictVo{FilterName: fullyDilutedValuationFilterHandler.FilterName(), Outcome: vo.FilterOutcomePassed}
	if coinProfile.MarketData == nil || coinProfile.MarketData.FullyDilutedValuationUsd == nil {
		verdict.Outcome, verdict.Reason = vo.FilterOutcomeNoData, "查不到完全稀釋估值"
		return verdict
	}

	fullyDilutedValuationUsd := *coinProfile.MarketData.FullyDilutedValuationUsd
	valuationText := domains.NewNumberDescriptionDomain(fullyDilutedValuationUsd).AsUsd()
	switch {
	case fullyDilutedValuationUsd.LessThan(fullyDilutedValuationFilterHandler.minimumFullyDilutedValuationUsd):
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = "完全稀釋估值 " + valuationText + "低於下限 " +
			domains.NewNumberDescriptionDomain(fullyDilutedValuationFilterHandler.minimumFullyDilutedValuationUsd).AsUsd()
	case fullyDilutedValuationUsd.GreaterThan(fullyDilutedValuationFilterHandler.maximumFullyDilutedValuationUsd):
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = "完全稀釋估值 " + valuationText + "高於上限 " +
			domains.NewNumberDescriptionDomain(fullyDilutedValuationFilterHandler.maximumFullyDilutedValuationUsd).AsUsd()
	}

	return verdict
}
