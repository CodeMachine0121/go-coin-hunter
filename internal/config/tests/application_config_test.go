package config_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadFallsBackToDefaults(t *testing.T) {
	for _, name := range []string{"SERVER_ADDRESS", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD",
		"POSTGRES_DATABASE", "POSTGRES_SSL_MODE", "BACKGROUND_JOBS_ENABLED", "DISCOVERY_WINDOW_HOURS",
		"DISCOVERY_EXCLUDED_COIN_SYMBOLS"} {
		t.Setenv(name, "")
	}

	applicationConfig := config.Load()

	assert.Equal(t, ":8080", applicationConfig.ServerAddress)
	assert.Equal(t, "host=localhost port=5432 user=postgres password=postgres dbname=go_coin_hunter sslmode=disable",
		applicationConfig.Database.DataSourceName())
	assert.True(t, applicationConfig.BackgroundJobsEnabled)
	assert.Equal(t, 72*time.Hour, applicationConfig.Discovery.Window)
	assert.Equal(t, []string{"BTC", "ETH", "BNB", "SOL", "XRP", "USDT", "USDC", "FDUSD", "DAI", "TUSD", "USDE"}, applicationConfig.Discovery.ExcludedCoinSymbols)
	assert.Equal(t, 50, applicationConfig.Discovery.ItemLimitPerSource)
	assert.Equal(t, 15*time.Second, applicationConfig.Discovery.SourceRequestTimeout)
}

func TestLoadReadsOverrides(t *testing.T) {
	t.Setenv("BACKGROUND_JOBS_ENABLED", "false")
	t.Setenv("DISCOVERY_WINDOW_HOURS", "24")
	t.Setenv("DISCOVERY_EXCLUDED_COIN_SYMBOLS", " BTC , ,DOGE")

	applicationConfig := config.Load()

	assert.False(t, applicationConfig.BackgroundJobsEnabled)
	assert.Equal(t, 24*time.Hour, applicationConfig.Discovery.Window)
	assert.Equal(t, []string{"BTC", "DOGE"}, applicationConfig.Discovery.ExcludedCoinSymbols)
}

func TestLoadReadsFilteringThresholds(t *testing.T) {
	t.Setenv("FILTER_MAXIMUM_TAX_RATE", "")
	t.Setenv("FILTER_MINIMUM_DAILY_VOLUME_USD", "2500000")
	t.Setenv("FILTER_MINIMUM_FDV_USD", "-1")
	t.Setenv("FILTER_UNLOCK_LOOKAHEAD_DAYS", "7")

	filteringConfig := config.Load().Filtering

	assert.Equal(t, "0.1", filteringConfig.MaximumTaxRate.String())
	assert.Equal(t, "2500000", filteringConfig.MinimumDailyVolumeUsd.String())
	assert.Equal(t, "10000000", filteringConfig.MinimumFullyDilutedValuationUsd.String())
	assert.Equal(t, "1000000000", filteringConfig.MaximumFullyDilutedValuationUsd.String())
	assert.Equal(t, "0.2", filteringConfig.MinimumCirculatingRatio.String())
	assert.Equal(t, 7*24*time.Hour, filteringConfig.UnlockLookahead)
	assert.Equal(t, "0.05", filteringConfig.MaximumUnlockRatioOfCirculating.String())
	assert.Equal(t, 20*time.Second, filteringConfig.SourceRequestTimeout)
	assert.Equal(t, 60*time.Second, filteringConfig.RoundBaseBudget)
	assert.Equal(t, 2*time.Second, filteringConfig.TokenSecurityRequestInterval)
}

func TestLoadInsightDefaults(t *testing.T) {
	for _, name := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "INSIGHT_MODEL", "INSIGHT_EFFORT",
		"INSIGHT_MAX_CONCURRENT_ANALYSES", "INSIGHT_MAX_COINS_PER_ROUND", "GOOGLE_NEWS_BASE_URL"} {
		t.Setenv(name, "")
	}

	insightConfig := config.Load().Insight

	assert.Equal(t, config.InsightConfig{
		Model: "claude-opus-5-5", Effort: "low", AnalysisTimeout: 120 * time.Second, MaximumConcurrentAnalyses: 3, MaximumCoinsPerRound: 20,
		GoogleNewsBaseUrl: "https://news.google.com", MaximumIntelligenceHeadlines: 20, MaximumNewsHeadlines: 10,
		NewsLookback: 72 * time.Hour, MaterialSourceTimeout: 15 * time.Second,
	}, insightConfig)
}

func TestLoadReadsInsightSettings(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("INSIGHT_MODEL", "")
	t.Setenv("INSIGHT_EFFORT", "medium")
	t.Setenv("INSIGHT_MAX_COINS_PER_ROUND", "0")

	insightConfig := config.Load().Insight

	assert.Equal(t, "test-key", insightConfig.AnthropicApiKey)
	assert.Equal(t, "claude-opus-5-5", insightConfig.Model)
	assert.Equal(t, "medium", insightConfig.Effort)
	assert.Equal(t, 120*time.Second, insightConfig.AnalysisTimeout)
	assert.Equal(t, 3, insightConfig.MaximumConcurrentAnalyses)
	assert.Equal(t, 20, insightConfig.MaximumCoinsPerRound)
	assert.Equal(t, "https://news.google.com", insightConfig.GoogleNewsBaseUrl)
}

func TestLoadIgnoresANonPositiveWindow(t *testing.T) {
	t.Setenv("DISCOVERY_WINDOW_HOURS", "-3")

	assert.Equal(t, 72*time.Hour, config.Load().Discovery.Window)
}

func TestLoadKeepsTheSourceLimitAndTimeoutFixed(t *testing.T) {
	t.Setenv("INFORMATION_SOURCE_ITEM_LIMIT", "500")
	t.Setenv("INFORMATION_SOURCE_TIMEOUT_SECONDS", "120")

	applicationConfig := config.Load()

	assert.Equal(t, 50, applicationConfig.Discovery.ItemLimitPerSource)
	assert.Equal(t, 15*time.Second, applicationConfig.Discovery.SourceRequestTimeout)
}
