package marketdata_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/marketdata"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveByPath answers by path, with a status code per path when one is given.
func serveByPath(t *testing.T, bodies map[string]string, statuses map[string]int) string {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if status, has := statuses[request.URL.Path]; has {
			writer.WriteHeader(status)
		}
		body, known := bodies[request.URL.Path]
		if !known {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// describe writes every figure as decimal text, so equal values compare equal whatever their internal exponent.
func describe(marketStructure vo.PerpetualMarketStructureVo) map[string]string {
	text := func(value *decimal.Decimal) string {
		if value == nil {
			return "unknown"
		}
		return value.String()
	}
	return map[string]string{
		"exchange": marketStructure.ExchangeName, "lastPrice": text(marketStructure.LastPrice), "priceChange": text(marketStructure.PriceChangeRatio24h),
		"quoteVolume": text(marketStructure.QuoteVolumeUsd24h), "fundingRate": text(marketStructure.FundingRate),
		"openInterest": text(marketStructure.OpenInterestUsd), "openInterestChange": text(marketStructure.OpenInterestChangeRatio24h),
		"fundingInterval": hours(marketStructure.FundingIntervalHours),
	}
}

func hours(value *int) string {
	if value == nil {
		return "unknown"
	}
	return strconv.Itoa(*value)
}

func intervalOf(hours int) *int {
	return &hours
}

func TestBinanceMarketStructure(t *testing.T) {
	baseUrl := serveByPath(t, map[string]string{
		"/fapi/v1/ticker/24hr":           `{"lastPrice":"0.0097","priceChangePercent":"6.002","quoteVolume":"67801301.99"}`,
		"/fapi/v1/premiumIndex":          `{"lastFundingRate":"0.00005"}`,
		"/fapi/v1/fundingInfo":           `[{"symbol":"PENGUUSDT","fundingIntervalHours":4},{"symbol":"PONSUSDT","fundingIntervalHours":1}]`,
		"/futures/data/openInterestHist": `[{"sumOpenInterestValue":"110","timestamp":2},{"sumOpenInterestValue":"100","timestamp":1}]`,
	}, nil)

	marketStructure, found, findError := marketdata.NewBinancePerpetualMarketStructureProxy(http.DefaultClient, baseUrl).FindMarketStructure(context.Background(), "PENGU")

	require.NoError(t, findError)
	assert.True(t, found)
	assert.Equal(t, describe(vo.PerpetualMarketStructureVo{ExchangeName: "幣安", LastPrice: amount("0.0097"), PriceChangeRatio24h: amount("0.06002"),
		QuoteVolumeUsd24h: amount("67801301.99"), FundingRate: amount("0.00005"), FundingIntervalHours: intervalOf(4), OpenInterestUsd: amount("110"),
		OpenInterestChangeRatio24h: amount("0.1")}), describe(marketStructure))
}

func TestBinanceMarketStructureWithThinHistory(t *testing.T) {
	baseUrl := serveByPath(t, map[string]string{
		"/fapi/v1/ticker/24hr":           `{"lastPrice":"1"}`,
		"/fapi/v1/premiumIndex":          `{}`,
		"/fapi/v1/fundingInfo":           `[{"symbol":"PENGUUSDT","fundingIntervalHours":4}]`,
		"/futures/data/openInterestHist": `[{"sumOpenInterestValue":"5","timestamp":1}]`,
	}, nil)

	marketStructure, _, findError := marketdata.NewBinancePerpetualMarketStructureProxy(http.DefaultClient, baseUrl).FindMarketStructure(context.Background(), "NEW")

	require.NoError(t, findError)
	// A contract Binance does not list among the adjusted ones settles every 8 hours.
	assert.Equal(t, describe(vo.PerpetualMarketStructureVo{ExchangeName: "幣安", LastPrice: amount("1"), FundingIntervalHours: intervalOf(8),
		OpenInterestUsd: amount("5")}), describe(marketStructure))
}

func TestBybitMarketStructure(t *testing.T) {
	baseUrl := serveByPath(t, map[string]string{
		"/v5/market/tickers":          `{"retCode":0,"result":{"list":[{"lastPrice":"0.0097","price24hPcnt":"0.06","turnover24h":"15698583.81","fundingRate":"0.00005","openInterestValue":"25038853.83"}]}}`,
		"/v5/market/open-interest":    `{"retCode":0,"result":{"list":[{"openInterest":"90","timestamp":"1791172800000"},{"openInterest":"100","timestamp":"1791086400000"}]}}`,
		"/v5/market/instruments-info": `{"retCode":0,"result":{"list":[{"fundingInterval":60}]}}`,
	}, nil)

	marketStructure, found, findError := marketdata.NewBybitPerpetualMarketStructureProxy(http.DefaultClient, baseUrl).FindMarketStructure(context.Background(), "PENGU")

	require.NoError(t, findError)
	assert.True(t, found)
	assert.Equal(t, describe(vo.PerpetualMarketStructureVo{ExchangeName: "Bybit", LastPrice: amount("0.0097"), PriceChangeRatio24h: amount("0.06"),
		QuoteVolumeUsd24h: amount("15698583.81"), FundingRate: amount("0.00005"), FundingIntervalHours: intervalOf(1), OpenInterestUsd: amount("25038853.83"),
		OpenInterestChangeRatio24h: amount("-0.1")}), describe(marketStructure))
}

func TestOkxMarketStructure(t *testing.T) {
	baseUrl := serveByPath(t, map[string]string{
		"/api/v5/market/ticker":        `{"code":"0","data":[{"last":"0.01","open24h":"0.008","volCcy24h":"2000000"}]}`,
		"/api/v5/public/funding-rate":  `{"code":"0","data":[{"fundingRate":"0.0001","fundingTime":"1791100800000","nextFundingTime":"1791115200000"}]}`,
		"/api/v5/public/open-interest": `{"code":"0","data":[{"oiUsd":"10083117.45"}]}`,
	}, nil)

	marketStructure, found, findError := marketdata.NewOkxPerpetualMarketStructureProxy(http.DefaultClient, baseUrl).FindMarketStructure(context.Background(), "PENGU")

	require.NoError(t, findError)
	assert.True(t, found)
	assert.Equal(t, describe(vo.PerpetualMarketStructureVo{ExchangeName: "OKX", LastPrice: amount("0.01"), PriceChangeRatio24h: amount("0.25"),
		QuoteVolumeUsd24h: amount("20000"), FundingRate: amount("0.0001"), FundingIntervalHours: intervalOf(4), OpenInterestUsd: amount("10083117.45")}), describe(marketStructure))
}

func TestOkxMarketStructureWithoutFundingOrOpenInterest(t *testing.T) {
	baseUrl := serveByPath(t, map[string]string{
		"/api/v5/market/ticker":        `{"code":"0","data":[{"last":"0.01"}]}`,
		"/api/v5/public/funding-rate":  `{"code":"0","data":[]}`,
		"/api/v5/public/open-interest": `{"code":"0","data":[]}`,
	}, nil)

	marketStructure, _, findError := marketdata.NewOkxPerpetualMarketStructureProxy(http.DefaultClient, baseUrl).FindMarketStructure(context.Background(), "PENGU")

	require.NoError(t, findError)
	assert.Equal(t, describe(vo.PerpetualMarketStructureVo{ExchangeName: "OKX", LastPrice: amount("0.01")}), describe(marketStructure))
}

func TestMarketStructureWithoutAContract(t *testing.T) {
	binanceUrl := serveByPath(t, map[string]string{"/fapi/v1/ticker/24hr": `{"code":-1121,"msg":"Invalid symbol."}`}, map[string]int{"/fapi/v1/ticker/24hr": http.StatusBadRequest})
	bybitUrl := serveByPath(t, map[string]string{"/v5/market/tickers": `{"retCode":10001,"retMsg":"params error"}`}, nil)
	bybitEmptyUrl := serveByPath(t, map[string]string{"/v5/market/tickers": `{"retCode":0,"result":{"list":[]}}`}, nil)
	okxUrl := serveByPath(t, map[string]string{"/api/v5/market/ticker": `{"code":"51001","data":[]}`}, nil)

	for name, find := range map[string]func() (bool, error){
		"binance": func() (bool, error) {
			_, found, findError := marketdata.NewBinancePerpetualMarketStructureProxy(http.DefaultClient, binanceUrl).FindMarketStructure(context.Background(), "X")
			return found, findError
		},
		"bybit": func() (bool, error) {
			_, found, findError := marketdata.NewBybitPerpetualMarketStructureProxy(http.DefaultClient, bybitUrl).FindMarketStructure(context.Background(), "X")
			return found, findError
		},
		"bybit empty": func() (bool, error) {
			_, found, findError := marketdata.NewBybitPerpetualMarketStructureProxy(http.DefaultClient, bybitEmptyUrl).FindMarketStructure(context.Background(), "X")
			return found, findError
		},
		"okx": func() (bool, error) {
			_, found, findError := marketdata.NewOkxPerpetualMarketStructureProxy(http.DefaultClient, okxUrl).FindMarketStructure(context.Background(), "X")
			return found, findError
		},
	} {
		t.Run(name, func(t *testing.T) {
			found, findError := find()

			assert.NoError(t, findError)
			assert.False(t, found)
		})
	}
}

func TestMarketStructureFailsOnBrokenSources(t *testing.T) {
	testCases := map[string]func() error{
		"binance ticker down": func() error {
			_, _, findError := marketdata.NewBinancePerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"binance funding down": func() error {
			_, _, findError := marketdata.NewBinancePerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{"/fapi/v1/ticker/24hr": `{}`}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"binance history down": func() error {
			_, _, findError := marketdata.NewBinancePerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{"/fapi/v1/ticker/24hr": `{}`, "/fapi/v1/premiumIndex": `{}`}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"bybit down": func() error {
			_, _, findError := marketdata.NewBybitPerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"bybit without a return code": func() error {
			_, _, findError := marketdata.NewBybitPerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{"/v5/market/tickers": `{}`}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"bybit history down": func() error {
			_, _, findError := marketdata.NewBybitPerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{"/v5/market/tickers": `{"retCode":0,"result":{"list":[{}]}}`}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"okx down": func() error {
			_, _, findError := marketdata.NewOkxPerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"okx refusing": func() error {
			_, _, findError := marketdata.NewOkxPerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{"/api/v5/market/ticker": `{"code":"50011","data":[]}`}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"okx funding down": func() error {
			_, _, findError := marketdata.NewOkxPerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{"/api/v5/market/ticker": `{"code":"0","data":[{}]}`}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
		"okx open interest down": func() error {
			_, _, findError := marketdata.NewOkxPerpetualMarketStructureProxy(http.DefaultClient, serveByPath(t, map[string]string{"/api/v5/market/ticker": `{"code":"0","data":[{}]}`, "/api/v5/public/funding-rate": `{"code":"0","data":[]}`}, nil)).FindMarketStructure(context.Background(), "X")
			return findError
		},
	}

	for name, fetch := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, fetch())
		})
	}
}

func TestBybitMarketStructureWithoutAFundingPeriod(t *testing.T) {
	baseUrl := serveByPath(t, map[string]string{
		"/v5/market/tickers":          `{"retCode":0,"result":{"list":[{"lastPrice":"1"}]}}`,
		"/v5/market/open-interest":    `{"retCode":0,"result":{"list":[]}}`,
		"/v5/market/instruments-info": `{"retCode":0,"result":{"list":[]}}`,
	}, nil)

	marketStructure, _, findError := marketdata.NewBybitPerpetualMarketStructureProxy(http.DefaultClient, baseUrl).FindMarketStructure(context.Background(), "PENGU")

	require.NoError(t, findError)
	assert.Equal(t, "unknown", hours(marketStructure.FundingIntervalHours))
}

func TestOkxMarketStructureWithoutTheNextFundingTime(t *testing.T) {
	baseUrl := serveByPath(t, map[string]string{
		"/api/v5/market/ticker":        `{"code":"0","data":[{"last":"0.01"}]}`,
		"/api/v5/public/funding-rate":  `{"code":"0","data":[{"fundingRate":"0.0001","fundingTime":"1791100800000"}]}`,
		"/api/v5/public/open-interest": `{"code":"0","data":[]}`,
	}, nil)

	marketStructure, _, findError := marketdata.NewOkxPerpetualMarketStructureProxy(http.DefaultClient, baseUrl).FindMarketStructure(context.Background(), "PENGU")

	require.NoError(t, findError)
	assert.Equal(t, "0.0001", marketStructure.FundingRate.String())
	assert.Equal(t, "unknown", hours(marketStructure.FundingIntervalHours))
}

func TestMarketStructureFailsWhenTheFundingPeriodCannotBeRead(t *testing.T) {
	binanceUrl := serveByPath(t, map[string]string{"/fapi/v1/ticker/24hr": `{}`, "/fapi/v1/premiumIndex": `{}`}, nil)
	_, _, binanceError := marketdata.NewBinancePerpetualMarketStructureProxy(http.DefaultClient, binanceUrl).FindMarketStructure(context.Background(), "PENGU")
	assert.Error(t, binanceError)

	bybitUrl := serveByPath(t, map[string]string{
		"/v5/market/tickers":       `{"retCode":0,"result":{"list":[{"lastPrice":"1"}]}}`,
		"/v5/market/open-interest": `{"retCode":0,"result":{"list":[]}}`,
	}, nil)
	_, _, bybitError := marketdata.NewBybitPerpetualMarketStructureProxy(http.DefaultClient, bybitUrl).FindMarketStructure(context.Background(), "PENGU")
	assert.Error(t, bybitError)
}
