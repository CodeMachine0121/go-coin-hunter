package marketdata

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"github.com/shopspring/decimal"
)

var percentDivisor = decimal.NewFromInt(100)

// binanceDefaultFundingIntervalHours is the period of every contract Binance's funding info does not list.
const binanceDefaultFundingIntervalHours = 8

// BinancePerpetualMarketStructureProxy reads Binance USDⓈ-M futures: the 24h ticker, the funding rate and its period, and
// 25 hourly open interest readings, so the change over the last day can be told.
type BinancePerpetualMarketStructureProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewBinancePerpetualMarketStructureProxy(httpClient *http.Client, baseUrl string) *BinancePerpetualMarketStructureProxy {
	return &BinancePerpetualMarketStructureProxy{httpClient: httpClient, baseUrl: baseUrl}
}

// FindMarketStructure treats Binance refusing the symbol (400) as "no such contract".
func (binancePerpetualMarketStructureProxy *BinancePerpetualMarketStructureProxy) FindMarketStructure(
	executionContext context.Context, coinSymbol string,
) (vo.PerpetualMarketStructureVo, bool, error) {
	contractSymbol := url.QueryEscape(coinSymbol + "USDT")
	ticker, tickerError := utilities.GetJson[binanceTicker24hWire](executionContext, binancePerpetualMarketStructureProxy.httpClient,
		binancePerpetualMarketStructureProxy.baseUrl+"/fapi/v1/ticker/24hr?symbol="+contractSymbol)
	httpStatusError := utilities.HttpStatusError{}
	if errors.As(tickerError, &httpStatusError) && httpStatusError.StatusCode == http.StatusBadRequest {
		return vo.PerpetualMarketStructureVo{}, false, nil
	}
	if tickerError != nil {
		return vo.PerpetualMarketStructureVo{}, false, tickerError
	}
	premiumIndex, premiumError := utilities.GetJson[binancePremiumIndexWire](executionContext, binancePerpetualMarketStructureProxy.httpClient,
		binancePerpetualMarketStructureProxy.baseUrl+"/fapi/v1/premiumIndex?symbol="+contractSymbol)
	if premiumError != nil {
		return vo.PerpetualMarketStructureVo{}, false, premiumError
	}
	fundingInfos, fundingInfoError := utilities.GetJson[[]binanceFundingInfoWire](executionContext, binancePerpetualMarketStructureProxy.httpClient,
		binancePerpetualMarketStructureProxy.baseUrl+"/fapi/v1/fundingInfo")
	if fundingInfoError != nil {
		return vo.PerpetualMarketStructureVo{}, false, fundingInfoError
	}
	openInterestHistory, historyError := utilities.GetJson[[]binanceOpenInterestHistoryWire](executionContext, binancePerpetualMarketStructureProxy.httpClient,
		fmt.Sprintf("%s/futures/data/openInterestHist?symbol=%s&period=1h&limit=25", binancePerpetualMarketStructureProxy.baseUrl, contractSymbol))
	if historyError != nil {
		return vo.PerpetualMarketStructureVo{}, false, historyError
	}

	marketStructure := vo.PerpetualMarketStructureVo{
		ExchangeName:      "幣安",
		LastPrice:         ticker.LastPrice.decimalOrNil(),
		QuoteVolumeUsd24h: ticker.QuoteVolume.decimalOrNil(),
		FundingRate:       premiumIndex.LastFundingRate.decimalOrNil(),
	}
	fundingIntervalHours := binanceDefaultFundingIntervalHours
	for _, fundingInfo := range fundingInfos {
		if fundingInfo.Symbol == coinSymbol+"USDT" && fundingInfo.FundingIntervalHours > 0 {
			fundingIntervalHours = fundingInfo.FundingIntervalHours
		}
	}
	marketStructure.FundingIntervalHours = &fundingIntervalHours
	if ticker.PriceChangePercent != nil {
		priceChangeRatio := ticker.PriceChangePercent.value.Div(percentDivisor)
		marketStructure.PriceChangeRatio24h = &priceChangeRatio
	}
	slices.SortFunc(openInterestHistory, func(left, right binanceOpenInterestHistoryWire) int {
		return cmp.Compare(left.Timestamp, right.Timestamp)
	})
	if len(openInterestHistory) > 0 {
		latest := openInterestHistory[len(openInterestHistory)-1]
		marketStructure.OpenInterestUsd = latest.SumOpenInterestValue.decimalOrNil()
	}
	if len(openInterestHistory) > 1 {
		marketStructure.OpenInterestChangeRatio24h = openInterestHistory[0].SumOpenInterestValue.changeRatioTo(
			openInterestHistory[len(openInterestHistory)-1].SumOpenInterestValue)
	}

	return marketStructure, true, nil
}
