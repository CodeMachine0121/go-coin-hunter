package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/config"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/analysis"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/marketdata"
	"github.com/CodeMachine0121/go-coin-hunter/internal/job"
	"github.com/stretchr/testify/assert"
)

// The insight rules prefer Binance, then Bybit, then OKX; the list order is that preference.
func TestPerpetualMarketStructureExchangesAreListedInPreferenceOrder(t *testing.T) {
	proxies := perpetualMarketStructureProxiesFor(http.DefaultClient, config.DiscoveryConfig{})

	assert.Len(t, proxies, 3)
	assert.IsType(t, &marketdata.BinancePerpetualMarketStructureProxy{}, proxies[0])
	assert.IsType(t, &marketdata.BybitPerpetualMarketStructureProxy{}, proxies[1])
	assert.IsType(t, &marketdata.OkxPerpetualMarketStructureProxy{}, proxies[2])
}

func TestVerdictsAreClampedIntoTheAgreedSafeRanges(t *testing.T) {
	huntVerdictPolicy := huntVerdictPolicyFor(config.VerdictConfig{MinimumBullishInsightStrength: 6, MinimumHuntBoardConfidence: 50})

	assert.Equal(t, 1, huntVerdictPolicy.MinimumLeverage)
	assert.Equal(t, 5, huntVerdictPolicy.MaximumLeverage)
	assert.Equal(t, "10", huntVerdictPolicy.MaximumPositionSizePercent.String())
	assert.Equal(t, "1", huntVerdictPolicy.MinimumStopLossPercent.String())
	assert.Equal(t, "50", huntVerdictPolicy.MaximumStopLossPercent.String())
	assert.Equal(t, "1", huntVerdictPolicy.MinimumTakeProfitPercent.String())
	assert.Equal(t, "200", huntVerdictPolicy.MaximumTakeProfitPercent.String())
	assert.Equal(t, 6, huntVerdictPolicy.MinimumBullishInsightStrength)
	assert.Equal(t, 50, huntVerdictPolicy.MinimumHuntBoardConfidence)
}

func TestEveryFilteringRuleIsPluggedIn(t *testing.T) {
	filterNames := []string{}
	for _, filterHandler := range filterHandlersFor(config.FilteringConfig{}) {
		filterNames = append(filterNames, filterHandler.FilterName())
	}

	assert.Equal(t, []string{"securityCheck", "liquidityThreshold", "fullyDilutedValuation", "circulatingRatio", "unlockSchedule",
		"perpetualContractListing", "priceChange", "openInterestChange", "fundingRateOverheat"}, filterNames)
}

func TestEachClaudeCapabilityKeepsItsOwnSettings(t *testing.T) {
	insightSettings, verdictSettings := claudeModelSettingsFor(config.ApplicationConfig{
		Insight: config.InsightConfig{Model: "insight-model", Effort: "low", AnalysisTimeout: 120 * time.Second},
		Verdict: config.VerdictConfig{Model: "verdict-model", Effort: "high", SynthesisTimeout: 180 * time.Second},
	})

	assert.Equal(t, analysis.ClaudeModelSettings{Model: "insight-model", Effort: "low", RequestTimeout: 120 * time.Second}, insightSettings)
	assert.Equal(t, analysis.ClaudeModelSettings{Model: "verdict-model", Effort: "high", RequestTimeout: 180 * time.Second}, verdictSettings)
}

func TestInsightShowsIntelligenceFromTheDiscoveryWindow(t *testing.T) {
	coinInsightPolicy := coinInsightPolicyFor(config.ApplicationConfig{
		Discovery: config.DiscoveryConfig{Window: 24 * time.Hour},
		Insight:   config.InsightConfig{NewsLookback: 72 * time.Hour, MaximumIntelligenceHeadlines: 20, MaximumNewsHeadlines: 10, MaterialSourceTimeout: 15 * time.Second, MaximumCoinsPerRound: 20, MaximumConcurrentAnalyses: 3},
	})

	assert.Equal(t, 24*time.Hour, coinInsightPolicy.IntelligenceWindow)
	assert.Equal(t, 72*time.Hour, coinInsightPolicy.NewsLookback)
	assert.Equal(t, 20, coinInsightPolicy.MaximumIntelligenceHeadline)
	assert.Equal(t, 10, coinInsightPolicy.MaximumNewsHeadlines)
	assert.Equal(t, 15*time.Second, coinInsightPolicy.SourceRequestTimeout)
	assert.Equal(t, 20, coinInsightPolicy.MaximumCoinsPerRound)
	assert.Equal(t, 3, coinInsightPolicy.MaximumConcurrentAnalyses)
}

func TestTheHuntRoundIsScheduledOnlyWhenSwitchedOn(t *testing.T) {
	testCases := []struct {
		name                  string
		backgroundJobsEnabled bool
		interval              time.Duration
		wantJobs              int
	}{
		{name: "on with an interval", backgroundJobsEnabled: true, interval: 4 * time.Hour, wantJobs: 1},
		{name: "a zero interval", backgroundJobsEnabled: true, interval: 0, wantJobs: 0},
		{name: "a negative interval", backgroundJobsEnabled: true, interval: -time.Hour, wantJobs: 0},
		{name: "background jobs switched off", backgroundJobsEnabled: false, interval: 4 * time.Hour, wantJobs: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			backgroundJobs := backgroundJobsFor(config.ApplicationConfig{BackgroundJobsEnabled: testCase.backgroundJobsEnabled,
				HuntPipelineInterval: testCase.interval}, applications{})

			assert.Len(t, backgroundJobs, testCase.wantJobs)
			if testCase.wantJobs == 1 {
				assert.IsType(t, &job.HuntPipelineJob{}, backgroundJobs[0])
			}
		})
	}
}
