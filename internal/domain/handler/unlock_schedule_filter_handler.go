package handler

import (
	"strconv"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// UnlockScheduleFilterHandler rejects coins about to release a large share of their circulating supply.
type UnlockScheduleFilterHandler struct {
	unlockLookahead                 time.Duration
	maximumUnlockRatioOfCirculating decimal.Decimal
}

func NewUnlockScheduleFilterHandler(
	unlockLookahead time.Duration, maximumUnlockRatioOfCirculating decimal.Decimal,
) *UnlockScheduleFilterHandler {
	return &UnlockScheduleFilterHandler{
		unlockLookahead:                 unlockLookahead,
		maximumUnlockRatioOfCirculating: maximumUnlockRatioOfCirculating,
	}
}

func (unlockScheduleFilterHandler *UnlockScheduleFilterHandler) FilterName() string {
	return "unlockSchedule"
}

// Evaluate sums the unlocks after the evaluation time up to and including the end of the lookahead; reaching the ratio rejects.
func (unlockScheduleFilterHandler *UnlockScheduleFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	verdict := vo.FilterVerdictVo{FilterName: unlockScheduleFilterHandler.FilterName(), Outcome: vo.FilterOutcomeNoData}
	if !coinProfile.UnlockScheduleKnown {
		verdict.Reason = "查不到解鎖時程"
		return verdict
	}
	if coinProfile.MarketData == nil || coinProfile.MarketData.CirculatingSupply == nil || !coinProfile.MarketData.CirculatingSupply.IsPositive() {
		verdict.Reason = "查不到流通量"
		return verdict
	}

	lookaheadEnd := coinProfile.EvaluatedAt.Add(unlockScheduleFilterHandler.unlockLookahead)
	upcomingUnlockAmount := decimal.Zero
	for _, unlockEvent := range coinProfile.UnlockEvents {
		if unlockEvent.UnlockAt.After(coinProfile.EvaluatedAt) && !unlockEvent.UnlockAt.After(lookaheadEnd) {
			upcomingUnlockAmount = upcomingUnlockAmount.Add(unlockEvent.Amount)
		}
	}

	unlockRatio := upcomingUnlockAmount.Div(*coinProfile.MarketData.CirculatingSupply)
	verdict.Outcome = vo.FilterOutcomePassed
	if unlockRatio.GreaterThanOrEqual(unlockScheduleFilterHandler.maximumUnlockRatioOfCirculating) {
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = strconv.Itoa(int(unlockScheduleFilterHandler.unlockLookahead.Hours()/24)) + " 天內解鎖 " +
			domains.NewNumberDescriptionDomain(upcomingUnlockAmount).AsTokenQuantity() + "，達流通量 " +
			domains.NewNumberDescriptionDomain(unlockRatio).AsPercentage() + "（門檻 " +
			domains.NewNumberDescriptionDomain(unlockScheduleFilterHandler.maximumUnlockRatioOfCirculating).AsPercentage() + "）"
	}

	return verdict
}
