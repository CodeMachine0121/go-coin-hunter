package handler

import (
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// LiquidityThresholdFilterHandler keeps coins that trade enough to enter and leave a position.
type LiquidityThresholdFilterHandler struct {
	minimumDailyVolumeUsd decimal.Decimal
}

func NewLiquidityThresholdFilterHandler(minimumDailyVolumeUsd decimal.Decimal) *LiquidityThresholdFilterHandler {
	return &LiquidityThresholdFilterHandler{minimumDailyVolumeUsd: minimumDailyVolumeUsd}
}

func (liquidityThresholdFilterHandler *LiquidityThresholdFilterHandler) FilterName() string {
	return "liquidityThreshold"
}

func (liquidityThresholdFilterHandler *LiquidityThresholdFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	verdict := vo.FilterVerdictVo{FilterName: liquidityThresholdFilterHandler.FilterName(), Outcome: vo.FilterOutcomePassed}
	if coinProfile.MarketData == nil || coinProfile.MarketData.DailyVolumeUsd == nil {
		verdict.Outcome, verdict.Reason = vo.FilterOutcomeNoData, "查不到 24 小時成交額"
		return verdict
	}

	dailyVolumeUsd := *coinProfile.MarketData.DailyVolumeUsd
	if dailyVolumeUsd.LessThan(liquidityThresholdFilterHandler.minimumDailyVolumeUsd) {
		verdict.Outcome = vo.FilterOutcomeRejected
		verdict.Reason = "24 小時成交額 " + domains.NewNumberDescriptionDomain(dailyVolumeUsd).AsUsd() +
			"低於門檻 " + domains.NewNumberDescriptionDomain(liquidityThresholdFilterHandler.minimumDailyVolumeUsd).AsUsd()
	}

	return verdict
}
