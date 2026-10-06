package handler_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/handler"
	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var evaluatedAt = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

func amount(text string) *decimal.Decimal {
	value := decimal.RequireFromString(text)
	return &value
}

func ethereumContract() *vo.TokenAddressVo {
	return &vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xabc"}
}

type handlerCase struct {
	name        string
	coinProfile vo.CoinProfileVo
	wantOutcome vo.FilterOutcomeVo
	wantReason  string
}

func runHandlerCases(t *testing.T, filterHandler domaininterface.ICoinCandidateFilterHandler, wantFilterName string, testCases []handlerCase) {
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			verdict := filterHandler.Evaluate(testCase.coinProfile)

			assert.Equal(t, wantFilterName, verdict.FilterName)
			assert.Equal(t, testCase.wantOutcome, verdict.Outcome)
			assert.Equal(t, testCase.wantReason, verdict.Reason)
		})
	}
}

func TestSecurityCheckFilterHandler(t *testing.T) {
	zero := amount("0")
	runHandlerCases(t, handler.NewSecurityCheckFilterHandler(decimal.RequireFromString("0.1")), "securityCheck", []handlerCase{
		{name: "no contract risk passes", coinProfile: vo.CoinProfileVo{ContractAddress: ethereumContract(),
			TokenSecurity: &vo.TokenSecurityVo{BuyTaxRate: zero, SellTaxRate: zero}}, wantOutcome: vo.FilterOutcomePassed},
		{name: "a honeypot is rejected", coinProfile: vo.CoinProfileVo{ContractAddress: ethereumContract(),
			TokenSecurity: &vo.TokenSecurityVo{IsHoneypot: true, CannotSell: true}}, wantOutcome: vo.FilterOutcomeRejected, wantReason: "蜜罐，無法賣出"},
		{name: "a sell tax of exactly ten percent passes", coinProfile: vo.CoinProfileVo{ContractAddress: ethereumContract(),
			TokenSecurity: &vo.TokenSecurityVo{SellTaxRate: amount("0.1")}}, wantOutcome: vo.FilterOutcomePassed},
		{name: "a buy tax of exactly ten percent passes", coinProfile: vo.CoinProfileVo{ContractAddress: ethereumContract(),
			TokenSecurity: &vo.TokenSecurityVo{BuyTaxRate: amount("0.1")}}, wantOutcome: vo.FilterOutcomePassed},
		{name: "a sell tax over ten percent is rejected", coinProfile: vo.CoinProfileVo{ContractAddress: ethereumContract(),
			TokenSecurity: &vo.TokenSecurityVo{SellTaxRate: amount("0.105")}}, wantOutcome: vo.FilterOutcomeRejected, wantReason: "賣出稅 10.5% 超過上限 10%"},
		{name: "every other finding is listed", coinProfile: vo.CoinProfileVo{ContractAddress: ethereumContract(),
			TokenSecurity: &vo.TokenSecurityVo{CannotSell: true, BuyTaxRate: amount("0.2"), IsMintable: true, CanFreezeHolders: true}},
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "無法賣出全部持幣；買入稅 20% 超過上限 10%；發行方仍可增發；發行方可凍結持有人資產"},
		{name: "no contract address is no data", coinProfile: vo.CoinProfileVo{}, wantOutcome: vo.FilterOutcomeNoData, wantReason: "沒有主流鏈上的合約位址"},
		{name: "no findings is no data", coinProfile: vo.CoinProfileVo{ContractAddress: ethereumContract()},
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到合約安全資料"},
		{name: "a skipped lookup says why", coinProfile: vo.CoinProfileVo{ContractAddress: ethereumContract(), TokenSecurityNotQueriedReason: "未上永續合約，未查詢安全資料"},
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "未上永續合約，未查詢安全資料"},
	})
}

func TestLiquidityThresholdFilterHandler(t *testing.T) {
	runHandlerCases(t, handler.NewLiquidityThresholdFilterHandler(decimal.NewFromInt(1_000_000)), "liquidityThreshold", []handlerCase{
		{name: "exactly at the threshold passes", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{DailyVolumeUsd: amount("1000000")}},
			wantOutcome: vo.FilterOutcomePassed},
		{name: "below the threshold is rejected", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{DailyVolumeUsd: amount("990000")}},
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "24 小時成交額 99 萬美元低於門檻 100 萬美元"},
		{name: "no volume is no data", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{}},
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到 24 小時成交額"},
		{name: "no market data is no data", coinProfile: vo.CoinProfileVo{}, wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到 24 小時成交額"},
	})
}

func TestFullyDilutedValuationFilterHandler(t *testing.T) {
	runHandlerCases(t, handler.NewFullyDilutedValuationFilterHandler(decimal.NewFromInt(10_000_000), decimal.NewFromInt(1_000_000_000)),
		"fullyDilutedValuation", []handlerCase{
			{name: "exactly at the maximum passes", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{FullyDilutedValuationUsd: amount("1000000000")}},
				wantOutcome: vo.FilterOutcomePassed},
			{name: "exactly at the minimum passes", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{FullyDilutedValuationUsd: amount("10000000")}},
				wantOutcome: vo.FilterOutcomePassed},
			{name: "below the minimum is rejected", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{FullyDilutedValuationUsd: amount("9000000")}},
				wantOutcome: vo.FilterOutcomeRejected, wantReason: "完全稀釋估值 900 萬美元低於下限 1,000 萬美元"},
			{name: "above the maximum is rejected", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{FullyDilutedValuationUsd: amount("5200000000")}},
				wantOutcome: vo.FilterOutcomeRejected, wantReason: "完全稀釋估值 52 億美元高於上限 10 億美元"},
			{name: "no valuation is no data", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{}},
				wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到完全稀釋估值"},
		})
}

func TestCirculatingRatioFilterHandler(t *testing.T) {
	runHandlerCases(t, handler.NewCirculatingRatioFilterHandler(decimal.RequireFromString("0.2")), "circulatingRatio", []handlerCase{
		{name: "exactly twenty percent passes", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{
			CirculatingSupply: amount("200000000"), MaxSupply: amount("1000000000"), TotalSupply: amount("300000000")}}, wantOutcome: vo.FilterOutcomePassed},
		{name: "nineteen percent is rejected", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{
			CirculatingSupply: amount("190000000"), MaxSupply: amount("1000000000")}}, wantOutcome: vo.FilterOutcomeRejected, wantReason: "流通比 19% 低於門檻 20%"},
		{name: "the maximum is used over the total when both exist", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{
			CirculatingSupply: amount("150000000"), MaxSupply: amount("1000000000"), TotalSupply: amount("500000000")}},
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "流通比 15% 低於門檻 20%"},
		{name: "no maximum falls back to the total", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{
			CirculatingSupply: amount("300000000"), TotalSupply: amount("1000000000")}}, wantOutcome: vo.FilterOutcomePassed},
		{name: "a zero maximum falls back to the total", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{
			CirculatingSupply: amount("100000000"), MaxSupply: amount("0"), TotalSupply: amount("1000000000")}},
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "流通比 10% 低於門檻 20%"},
		{name: "no supply figures is no data", coinProfile: vo.CoinProfileVo{MarketData: &vo.CoinMarketDataVo{CirculatingSupply: amount("1")}},
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到最大供給量與總供給量"},
		{name: "no circulating supply is no data", coinProfile: vo.CoinProfileVo{}, wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到流通量"},
	})
}

func TestUnlockScheduleFilterHandler(t *testing.T) {
	billion := &vo.CoinMarketDataVo{CirculatingSupply: amount("1000000000")}
	unlockOn := func(days int, tokens string) vo.TokenUnlockEventVo {
		return vo.TokenUnlockEventVo{UnlockAt: evaluatedAt.Add(time.Duration(days) * 24 * time.Hour), Amount: decimal.RequireFromString(tokens)}
	}
	runHandlerCases(t, handler.NewUnlockScheduleFilterHandler(14*24*time.Hour, decimal.RequireFromString("0.05")), "unlockSchedule", []handlerCase{
		{name: "a small upcoming unlock passes", coinProfile: vo.CoinProfileVo{EvaluatedAt: evaluatedAt, MarketData: billion, UnlockScheduleKnown: true,
			UnlockEvents: []vo.TokenUnlockEventVo{unlockOn(3, "10000000")}}, wantOutcome: vo.FilterOutcomePassed},
		{name: "exactly five percent is rejected", coinProfile: vo.CoinProfileVo{EvaluatedAt: evaluatedAt, MarketData: billion, UnlockScheduleKnown: true,
			UnlockEvents: []vo.TokenUnlockEventVo{unlockOn(3, "20000000"), unlockOn(14, "30000000")}},
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "14 天內解鎖 5,000 萬，達流通量 5%（門檻 5%）"},
		{name: "an unlock on day fifteen is outside", coinProfile: vo.CoinProfileVo{EvaluatedAt: evaluatedAt, MarketData: billion, UnlockScheduleKnown: true,
			UnlockEvents: []vo.TokenUnlockEventVo{unlockOn(15, "200000000")}}, wantOutcome: vo.FilterOutcomePassed},
		{name: "past unlocks do not count", coinProfile: vo.CoinProfileVo{EvaluatedAt: evaluatedAt, MarketData: billion, UnlockScheduleKnown: true,
			UnlockEvents: []vo.TokenUnlockEventVo{unlockOn(-1, "200000000"), unlockOn(0, "200000000")}}, wantOutcome: vo.FilterOutcomePassed},
		{name: "an unknown schedule is no data", coinProfile: vo.CoinProfileVo{MarketData: billion}, wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到解鎖時程"},
		{name: "an unknown circulating supply is no data", coinProfile: vo.CoinProfileVo{UnlockScheduleKnown: true},
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到流通量"},
		{name: "a zero circulating supply is no data", coinProfile: vo.CoinProfileVo{UnlockScheduleKnown: true, MarketData: &vo.CoinMarketDataVo{CirculatingSupply: amount("0")}},
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到流通量"},
	})
}

func TestPerpetualContractListingFilterHandler(t *testing.T) {
	runHandlerCases(t, handler.NewPerpetualContractListingFilterHandler(), "perpetualContractListing", []handlerCase{
		{name: "one exchange is enough", coinProfile: vo.CoinProfileVo{PerpetualContractExchanges: []string{"Bybit"}},
			wantOutcome: vo.FilterOutcomePassed, wantReason: "已上 Bybit USDT 永續合約"},
		{name: "no exchange is rejected", coinProfile: vo.CoinProfileVo{}, wantOutcome: vo.FilterOutcomeRejected, wantReason: "幣安、Bybit、OKX 皆無 USDT 永續合約"},
	})
}

func withMarketStructure(marketStructure vo.PerpetualMarketStructureVo) vo.CoinProfileVo {
	return vo.CoinProfileVo{MarketStructure: &marketStructure}
}

func TestPriceChangeFilterHandler(t *testing.T) {
	runHandlerCases(t, handler.NewPriceChangeFilterHandler(decimal.RequireFromString("-0.1"), decimal.RequireFromString("0.6")), "priceChange", []handlerCase{
		{name: "a moderate rise passes", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{PriceChangeRatio24h: amount("0.12")}),
			wantOutcome: vo.FilterOutcomePassed},
		{name: "a fall of exactly ten percent passes", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{PriceChangeRatio24h: amount("-0.1")}),
			wantOutcome: vo.FilterOutcomePassed},
		{name: "a fall over ten percent is rejected", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{PriceChangeRatio24h: amount("-0.105")}),
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "24 小時跌幅 10.5% 超過下限 10%"},
		{name: "a rise of exactly sixty percent passes", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{PriceChangeRatio24h: amount("0.6")}),
			wantOutcome: vo.FilterOutcomePassed},
		{name: "a rise over sixty percent is rejected", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{PriceChangeRatio24h: amount("0.61")}),
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "24 小時漲幅 61% 超過上限 60%"},
		{name: "no market structure is no data", coinProfile: vo.CoinProfileVo{}, wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到永續合約市場結構"},
		{name: "no price change is no data", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{}),
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到 24 小時漲跌幅"},
	})
}

func TestOpenInterestChangeFilterHandler(t *testing.T) {
	runHandlerCases(t, handler.NewOpenInterestChangeFilterHandler(decimal.RequireFromString("-0.1")), "openInterestChange", []handlerCase{
		{name: "growing open interest passes", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{OpenInterestChangeRatio24h: amount("0.25")}),
			wantOutcome: vo.FilterOutcomePassed},
		{name: "a drop of exactly ten percent passes", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{OpenInterestChangeRatio24h: amount("-0.1")}),
			wantOutcome: vo.FilterOutcomePassed},
		{name: "a drop over ten percent is rejected", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{OpenInterestChangeRatio24h: amount("-0.11")}),
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "持倉量 24 小時減少 11% 超過下限 10%"},
		{name: "no market structure is no data", coinProfile: vo.CoinProfileVo{}, wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到永續合約市場結構"},
		{name: "no open interest change is no data", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{}),
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到持倉量 24 小時變化"},
	})
}

func TestFundingRateOverheatFilterHandler(t *testing.T) {
	runHandlerCases(t, handler.NewFundingRateOverheatFilterHandler(decimal.RequireFromString("0.001")), "fundingRateOverheat", []handlerCase{
		{name: "an ordinary rate passes", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{FundingRate: amount("0.0001")}),
			wantOutcome: vo.FilterOutcomePassed},
		{name: "exactly the ceiling passes", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{FundingRate: amount("0.001")}),
			wantOutcome: vo.FilterOutcomePassed},
		{name: "over the ceiling is rejected", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{FundingRate: amount("0.0011")}),
			wantOutcome: vo.FilterOutcomeRejected, wantReason: "資金費率 0.11% 高於上限 0.1%"},
		{name: "a negative rate passes", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{FundingRate: amount("-0.003")}),
			wantOutcome: vo.FilterOutcomePassed},
		{name: "no market structure is no data", coinProfile: vo.CoinProfileVo{}, wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到永續合約市場結構"},
		{name: "no funding rate is no data", coinProfile: withMarketStructure(vo.PerpetualMarketStructureVo{}),
			wantOutcome: vo.FilterOutcomeNoData, wantReason: "查不到資金費率"},
	})
}
