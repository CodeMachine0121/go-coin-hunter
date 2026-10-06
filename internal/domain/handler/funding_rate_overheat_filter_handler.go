package handler

import (
	"strconv"
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// fundingComparisonHours is the period every funding rate is scaled to before it meets the ceiling: most contracts settle
// every 8 hours, so a 1-hour contract paying a modest-looking rate is really eight times as crowded.
const fundingComparisonHours = 8

// FundingRateOverheatFilterHandler keeps coins whose longs are not already crowded; a negative rate never rejects.
type FundingRateOverheatFilterHandler struct {
	maximumEightHourFundingRate decimal.Decimal
}

func NewFundingRateOverheatFilterHandler(maximumEightHourFundingRate decimal.Decimal) *FundingRateOverheatFilterHandler {
	return &FundingRateOverheatFilterHandler{maximumEightHourFundingRate: maximumEightHourFundingRate}
}

func (fundingRateOverheatFilterHandler *FundingRateOverheatFilterHandler) FilterName() string {
	return "fundingRateOverheat"
}

// Evaluate scales the rate to 8 hours by the contract's funding period; an unknown period is taken as the usual 8 hours.
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
	fundingIntervalHours := fundingComparisonHours
	if coinProfile.MarketStructure.FundingIntervalHours != nil && *coinProfile.MarketStructure.FundingIntervalHours > 0 {
		fundingIntervalHours = *coinProfile.MarketStructure.FundingIntervalHours
	}
	eightHourFundingRate := fundingRate.Mul(decimal.NewFromInt(fundingComparisonHours)).Div(decimal.NewFromInt(int64(fundingIntervalHours)))
	verdict.Outcome = vo.FilterOutcomePassed
	if eightHourFundingRate.GreaterThan(fundingRateOverheatFilterHandler.maximumEightHourFundingRate) {
		verdict.Outcome = vo.FilterOutcomeRejected
		ceiling := " 高於上限 " + domains.NewNumberDescriptionDomain(fundingRateOverheatFilterHandler.maximumEightHourFundingRate).AsPercentage()
		verdict.Reason = "資金費率 " + domains.NewNumberDescriptionDomain(fundingRate).AsPercentage() + ceiling
		if fundingIntervalHours != fundingComparisonHours {
			verdict.Reason = "資金費率每 " + strconv.Itoa(fundingIntervalHours) + " 小時 " + domains.NewNumberDescriptionDomain(fundingRate).AsPercentage() +
				"，折合每 8 小時 " + domains.NewNumberDescriptionDomain(eightHourFundingRate).AsPercentage() + "，" + strings.TrimPrefix(ceiling, " ")
		}
	}

	return verdict
}
