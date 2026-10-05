package marketdata

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const okxCryptoInstrumentCategory = "1"

type OkxPerpetualContractListingProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewOkxPerpetualContractListingProxy(httpClient *http.Client, baseUrl string) *OkxPerpetualContractListingProxy {
	return &OkxPerpetualContractListingProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (okxPerpetualContractListingProxy *OkxPerpetualContractListingProxy) ExchangeName() string {
	return "OKX"
}

// FindUsdtPerpetualCoinSymbols reads the base coin from the instrument family, as in PENGU-USDT.
func (okxPerpetualContractListingProxy *OkxPerpetualContractListingProxy) FindUsdtPerpetualCoinSymbols(
	executionContext context.Context,
) (map[string]bool, error) {
	instruments, fetchError := getJson[okxInstrumentsWire](executionContext, okxPerpetualContractListingProxy.httpClient,
		okxPerpetualContractListingProxy.baseUrl+"/api/v5/public/instruments?instType=SWAP")
	if fetchError != nil {
		return nil, fetchError
	}
	if instruments.Code != "0" || instruments.Data == nil {
		return nil, fmt.Errorf("okx instruments answered %q without a contract list: %s", instruments.Code, instruments.Message)
	}

	coinSymbols := map[string]bool{}
	for _, contract := range *instruments.Data {
		baseCoin, quoteCoin, paired := strings.Cut(contract.InstrumentFamily, "-")
		if paired && quoteCoin == "USDT" && contract.SettleCurrency == "USDT" && contract.State == "live" &&
			contract.InstrumentCategory == okxCryptoInstrumentCategory {
			coinSymbols[strings.ToUpper(baseCoin)] = true
		}
	}

	return coinSymbols, nil
}
