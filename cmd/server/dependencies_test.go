package main

import (
	"net/http"
	"testing"

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
