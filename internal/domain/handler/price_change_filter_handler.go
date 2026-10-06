package handler

import (
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// PriceChangeFilterHandler keeps coins whose last day neither fell like a falling knife nor ran too far to chase.
type PriceChangeFilterHandler struct {
	minimumPriceChangeRatio decimal.Decimal
	maximumPriceChangeRatio decimal.Decimal
}

func NewPriceChangeFilterHandler(minimumPriceChangeRatio decimal.Decimal, maximumPriceChangeRatio decimal.Decimal) *PriceChangeFilterHandler {
	return &PriceChangeFilterHandler{minimumPriceChangeRatio: minimumPriceChangeRatio, maximumPriceChangeRatio: maximumPriceChangeRatio}
}

func (priceChangeFilterHandler *PriceChangeFilterHandler) FilterName() string {
	return "priceChange"
}

func (priceChangeFilterHandler *PriceChangeFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	verdict := vo.FilterVerdictVo{FilterName: priceChangeFilterHandler.FilterName(), Outcome: vo.FilterOutcomeNoData}
	if coinProfile.MarketStructure == nil {
		verdict.Reason = domains.MarketStructureUnknownReason
		return verdict
	}
	if coinProfile.MarketStructure.PriceChangeRatio24h == nil {
		verdict.Reason = "查不到 24 小時漲跌幅"
		return verdict
	}

	priceChangeRatio := *coinProfile.MarketStructure.PriceChangeRatio24h
	verdict.Outcome = vo.FilterOutcomePassed
	switch {
	case priceChangeRatio.LessThan(priceChangeFilterHandler.minimumPriceChangeRatio):
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = "24 小時跌幅 " + domains.NewNumberDescriptionDomain(priceChangeRatio.Neg()).AsPercentage() +
			" 超過下限 " + domains.NewNumberDescriptionDomain(priceChangeFilterHandler.minimumPriceChangeRatio.Neg()).AsPercentage()
	case priceChangeRatio.GreaterThan(priceChangeFilterHandler.maximumPriceChangeRatio):
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = "24 小時漲幅 " + domains.NewNumberDescriptionDomain(priceChangeRatio).AsPercentage() +
			" 超過上限 " + domains.NewNumberDescriptionDomain(priceChangeFilterHandler.maximumPriceChangeRatio).AsPercentage()
	}

	return verdict
}
