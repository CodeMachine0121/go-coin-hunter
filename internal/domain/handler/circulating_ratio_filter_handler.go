package handler

import (
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// CirculatingRatioFilterHandler rejects low-float coins, whose locked supply is future selling pressure.
type CirculatingRatioFilterHandler struct {
	minimumCirculatingRatio decimal.Decimal
}

func NewCirculatingRatioFilterHandler(minimumCirculatingRatio decimal.Decimal) *CirculatingRatioFilterHandler {
	return &CirculatingRatioFilterHandler{minimumCirculatingRatio: minimumCirculatingRatio}
}

func (circulatingRatioFilterHandler *CirculatingRatioFilterHandler) FilterName() string {
	return "circulatingRatio"
}

// Evaluate divides by the maximum supply, or by the total supply when there is no maximum.
func (circulatingRatioFilterHandler *CirculatingRatioFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	verdict := vo.FilterVerdictVo{FilterName: circulatingRatioFilterHandler.FilterName(), Outcome: vo.FilterOutcomeNoData}
	if coinProfile.MarketData == nil || coinProfile.MarketData.CirculatingSupply == nil {
		verdict.Reason = "查不到流通量"
		return verdict
	}

	marketData := coinProfile.MarketData
	denominator := decimal.Zero
	if marketData.TotalSupply != nil {
		denominator = *marketData.TotalSupply
	}
	if marketData.MaxSupply != nil && marketData.MaxSupply.IsPositive() {
		denominator = *marketData.MaxSupply
	}
	if !denominator.IsPositive() {
		verdict.Reason = "查不到最大供給量與總供給量"
		return verdict
	}

	circulatingRatio := marketData.CirculatingSupply.Div(denominator)
	verdict.Outcome = vo.FilterOutcomePassed
	if circulatingRatio.LessThan(circulatingRatioFilterHandler.minimumCirculatingRatio) {
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = "流通比 " + domains.NewNumberDescriptionDomain(circulatingRatio).AsPercentage() +
			" 低於門檻 " + domains.NewNumberDescriptionDomain(circulatingRatioFilterHandler.minimumCirculatingRatio).AsPercentage()
	}

	return verdict
}
