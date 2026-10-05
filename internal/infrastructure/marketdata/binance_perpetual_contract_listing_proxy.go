package marketdata

import (
	"context"
	"fmt"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"net/http"
	"strings"
)

type BinancePerpetualContractListingProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewBinancePerpetualContractListingProxy(httpClient *http.Client, baseUrl string) *BinancePerpetualContractListingProxy {
	return &BinancePerpetualContractListingProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (binancePerpetualContractListingProxy *BinancePerpetualContractListingProxy) ExchangeName() string {
	return "幣安"
}

// FindUsdtPerpetualCoinSymbols keeps trading USDT perpetuals of crypto assets; tradfi perpetuals have their own contract type.
func (binancePerpetualContractListingProxy *BinancePerpetualContractListingProxy) FindUsdtPerpetualCoinSymbols(
	executionContext context.Context,
) (map[string]bool, error) {
	exchangeInformation, fetchError := utilities.GetJson[binanceExchangeInformationWire](executionContext,
		binancePerpetualContractListingProxy.httpClient, binancePerpetualContractListingProxy.baseUrl+"/fapi/v1/exchangeInfo")
	if fetchError != nil {
		return nil, fetchError
	}
	if exchangeInformation.Symbols == nil {
		return nil, fmt.Errorf("binance exchange information answered without a symbol list")
	}

	coinSymbols := map[string]bool{}
	for _, contract := range *exchangeInformation.Symbols {
		if contract.ContractType == "PERPETUAL" && contract.QuoteAsset == "USDT" && contract.Status == "TRADING" {
			coinSymbols[strings.ToUpper(contract.BaseAsset)] = true
		}
	}

	return coinSymbols, nil
}
