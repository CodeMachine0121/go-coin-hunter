package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/config"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/marketdata"
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
