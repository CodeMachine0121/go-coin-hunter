package marketdata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"github.com/shopspring/decimal"
)

// okxInstrumentMissingCode is OKX saying the instrument does not exist.
const okxInstrumentMissingCode = "51001"

// millisecondsPerHour turns the gap between two funding times into the funding period.
var millisecondsPerHour = decimal.NewFromInt(3_600_000)

// OkxPerpetualMarketStructureProxy reads OKX's swap ticker, funding rate and current open interest; OKX keeps no free
// open interest history, so the day's change stays unknown.
type OkxPerpetualMarketStructureProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewOkxPerpetualMarketStructureProxy(httpClient *http.Client, baseUrl string) *OkxPerpetualMarketStructureProxy {
	return &OkxPerpetualMarketStructureProxy{httpClient: httpClient, baseUrl: baseUrl}
}

// FindMarketStructure derives the day's change from the 24h open and the quote volume from coin volume × last price.
func (okxPerpetualMarketStructureProxy *OkxPerpetualMarketStructureProxy) FindMarketStructure(
	executionContext context.Context, coinSymbol string,
) (vo.PerpetualMarketStructureVo, bool, error) {
	instrumentID := url.QueryEscape(coinSymbol + "-USDT-SWAP")
	tickers, tickerError := utilities.GetJson[okxResponseWire[okxTickerWire]](executionContext, okxPerpetualMarketStructureProxy.httpClient,
		okxPerpetualMarketStructureProxy.baseUrl+"/api/v5/market/ticker?instId="+instrumentID)
	if tickerError != nil {
		return vo.PerpetualMarketStructureVo{}, false, tickerError
	}
	if tickers.Code == okxInstrumentMissingCode {
		return vo.PerpetualMarketStructureVo{}, false, nil
	}
	if tickers.Code != "0" || len(tickers.Data) == 0 {
		return vo.PerpetualMarketStructureVo{}, false, fmt.Errorf("okx ticker answered %q without a ticker", tickers.Code)
	}
	fundingRates, fundingError := utilities.GetJson[okxResponseWire[okxFundingRateWire]](executionContext, okxPerpetualMarketStructureProxy.httpClient,
		okxPerpetualMarketStructureProxy.baseUrl+"/api/v5/public/funding-rate?instId="+instrumentID)
	if fundingError != nil {
		return vo.PerpetualMarketStructureVo{}, false, fundingError
	}
	openInterests, openInterestError := utilities.GetJson[okxResponseWire[okxOpenInterestWire]](executionContext, okxPerpetualMarketStructureProxy.httpClient,
		okxPerpetualMarketStructureProxy.baseUrl+"/api/v5/public/open-interest?instType=SWAP&instId="+instrumentID)
	if openInterestError != nil {
		return vo.PerpetualMarketStructureVo{}, false, openInterestError
	}

	ticker := tickers.Data[0]
	marketStructure := vo.PerpetualMarketStructureVo{ExchangeName: "OKX", LastPrice: ticker.Last.decimalOrNil()}
	marketStructure.PriceChangeRatio24h = ticker.Open24h.changeRatioTo(ticker.Last)
	if ticker.VolumeCurrency24h != nil && ticker.Last != nil {
		quoteVolumeUsd := ticker.VolumeCurrency24h.value.Mul(ticker.Last.value)
		marketStructure.QuoteVolumeUsd24h = &quoteVolumeUsd
	}
	if len(fundingRates.Data) > 0 {
		fundingRate := fundingRates.Data[0]
		marketStructure.FundingRate = fundingRate.FundingRate.decimalOrNil()
		if fundingRate.FundingTime != nil && fundingRate.NextFundingTime != nil {
			fundingIntervalHours := int(fundingRate.NextFundingTime.value.Sub(fundingRate.FundingTime.value).Div(millisecondsPerHour).IntPart())
			if fundingIntervalHours > 0 {
				marketStructure.FundingIntervalHours = &fundingIntervalHours
			}
		}
	}
	if len(openInterests.Data) > 0 {
		marketStructure.OpenInterestUsd = openInterests.Data[0].OpenInterestUsd.decimalOrNil()
	}

	return marketStructure, true, nil
}
