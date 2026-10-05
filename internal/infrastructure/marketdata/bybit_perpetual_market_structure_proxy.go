package marketdata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
)

// BybitPerpetualMarketStructureProxy reads Bybit's linear ticker and 25 hourly open interest readings.
type BybitPerpetualMarketStructureProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewBybitPerpetualMarketStructureProxy(httpClient *http.Client, baseUrl string) *BybitPerpetualMarketStructureProxy {
	return &BybitPerpetualMarketStructureProxy{httpClient: httpClient, baseUrl: baseUrl}
}

// FindMarketStructure treats Bybit answering with an error code or no ticker as "no such contract".
func (bybitPerpetualMarketStructureProxy *BybitPerpetualMarketStructureProxy) FindMarketStructure(
	executionContext context.Context, coinSymbol string,
) (vo.PerpetualMarketStructureVo, bool, error) {
	contractSymbol := url.QueryEscape(coinSymbol + "USDT")
	tickers, tickersError := utilities.GetJson[bybitTickersWire](executionContext, bybitPerpetualMarketStructureProxy.httpClient,
		bybitPerpetualMarketStructureProxy.baseUrl+"/v5/market/tickers?category=linear&symbol="+contractSymbol)
	if tickersError != nil {
		return vo.PerpetualMarketStructureVo{}, false, tickersError
	}
	if tickers.ReturnCode == nil {
		return vo.PerpetualMarketStructureVo{}, false, fmt.Errorf("bybit tickers answered without a return code")
	}
	if *tickers.ReturnCode != 0 || len(tickers.Result.List) == 0 {
		return vo.PerpetualMarketStructureVo{}, false, nil
	}
	openInterest, openInterestError := utilities.GetJson[bybitOpenInterestWire](executionContext, bybitPerpetualMarketStructureProxy.httpClient,
		bybitPerpetualMarketStructureProxy.baseUrl+"/v5/market/open-interest?category=linear&intervalTime=1h&limit=25&symbol="+contractSymbol)
	if openInterestError != nil {
		return vo.PerpetualMarketStructureVo{}, false, openInterestError
	}

	ticker := tickers.Result.List[0]
	marketStructure := vo.PerpetualMarketStructureVo{
		ExchangeName:        "Bybit",
		LastPrice:           ticker.LastPrice.decimalOrNil(),
		PriceChangeRatio24h: ticker.Price24hPercent.decimalOrNil(),
		QuoteVolumeUsd24h:   ticker.Turnover24h.decimalOrNil(),
		FundingRate:         ticker.FundingRate.decimalOrNil(),
		OpenInterestUsd:     ticker.OpenInterestValue.decimalOrNil(),
	}
	readings := openInterest.Result.List
	// Readings arrive newest first; they are put oldest first so the change runs forward in time.
	slices.SortFunc(readings, func(left, right bybitOpenInterestReadingWire) int {
		return left.Timestamp.value.Cmp(right.Timestamp.value)
	})
	if len(readings) > 1 {
		marketStructure.OpenInterestChangeRatio24h = readings[0].OpenInterest.changeRatioTo(readings[len(readings)-1].OpenInterest)
	}

	return marketStructure, true, nil
}
