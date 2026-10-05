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
		"DISCOVERY_EXCLUDED_COIN_SYMBOLS", "INFORMATION_SOURCE_ITEM_LIMIT", "INFORMATION_SOURCE_TIMEOUT_SECONDS"} {
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
	t.Setenv("INFORMATION_SOURCE_ITEM_LIMIT", "-3")

	applicationConfig := config.Load()

	assert.False(t, applicationConfig.BackgroundJobsEnabled)
	assert.Equal(t, 24*time.Hour, applicationConfig.Discovery.Window)
	assert.Equal(t, []string{"BTC", "DOGE"}, applicationConfig.Discovery.ExcludedCoinSymbols)
	assert.Equal(t, 50, applicationConfig.Discovery.ItemLimitPerSource)
}
