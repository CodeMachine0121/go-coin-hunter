package marketdata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// bybitCryptoSymbolTypes are the symbol types of crypto-native contracts.
var bybitCryptoSymbolTypes = map[string]bool{"": true, "innovation": true}

type BybitPerpetualContractListingProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewBybitPerpetualContractListingProxy(httpClient *http.Client, baseUrl string) *BybitPerpetualContractListingProxy {
	return &BybitPerpetualContractListingProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (bybitPerpetualContractListingProxy *BybitPerpetualContractListingProxy) ExchangeName() string {
	return "Bybit"
}

// FindUsdtPerpetualCoinSymbols follows the page cursor until the list ends.
func (bybitPerpetualContractListingProxy *BybitPerpetualContractListingProxy) FindUsdtPerpetualCoinSymbols(
	executionContext context.Context,
) (map[string]bool, error) {
	coinSymbols := map[string]bool{}
	cursor := ""
	for {
		instruments, fetchError := getJson[bybitInstrumentsWire](executionContext, bybitPerpetualContractListingProxy.httpClient,
			bybitPerpetualContractListingProxy.baseUrl+"/v5/market/instruments-info?category=linear&limit=1000&cursor="+url.QueryEscape(cursor))
		if fetchError != nil {
			return nil, fetchError
		}
		if instruments.ReturnCode == nil || *instruments.ReturnCode != 0 || instruments.Result.List == nil {
			return nil, fmt.Errorf("bybit instruments answered without a contract list: %s", instruments.ReturnMessage)
		}
		for _, contract := range *instruments.Result.List {
			if contract.ContractType == "LinearPerpetual" && contract.QuoteCoin == "USDT" && contract.Status == "Trading" &&
				bybitCryptoSymbolTypes[contract.SymbolType] {
				coinSymbols[strings.ToUpper(contract.BaseCoin)] = true
			}
		}
		if instruments.Result.NextPageCursor == "" || instruments.Result.NextPageCursor == cursor {
			return coinSymbols, nil
		}
		cursor = instruments.Result.NextPageCursor
	}
}
