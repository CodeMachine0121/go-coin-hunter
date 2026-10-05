package informationsource

import (
	"cmp"
	"context"
	"fmt"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"net/http"
	"slices"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// Delivery contracts are re-listed every quarter for old coins, so only perpetual kinds say anything about a new listing;
// the tradfi kind is a stock or commodity, not a coin.
const (
	binanceCryptoPerpetualContractType      = "PERPETUAL"
	binanceTraditionalPerpetualContractType = "TRADIFI_PERPETUAL"
)

// BinancePerpetualContractInformationSourceProxy reads the newest contracts on Binance USDⓈ-M futures.
type BinancePerpetualContractInformationSourceProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewBinancePerpetualContractInformationSourceProxy(httpClient *http.Client, baseUrl string) *BinancePerpetualContractInformationSourceProxy {
	return &BinancePerpetualContractInformationSourceProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (binancePerpetualContractInformationSourceProxy *BinancePerpetualContractInformationSourceProxy) SourceName() string {
	return "binancePerpetualContract"
}

// FetchInformationItems returns the most recently onboarded perpetual contracts; tradfi ones are kept but marked as traditional assets.
func (binancePerpetualContractInformationSourceProxy *BinancePerpetualContractInformationSourceProxy) FetchInformationItems(
	executionContext context.Context, itemLimit int,
) ([]vo.InformationItemVo, error) {
	exchangeInformation, fetchError := utilities.GetJson[binanceExchangeInformationWire](executionContext,
		binancePerpetualContractInformationSourceProxy.httpClient,
		binancePerpetualContractInformationSourceProxy.baseUrl+"/fapi/v1/exchangeInfo")
	if fetchError != nil {
		return nil, fetchError
	}
	if exchangeInformation.Symbols == nil {
		return nil, fmt.Errorf("binance exchange information answered without a symbol list")
	}

	contractSymbols := slices.DeleteFunc(slices.Clone(exchangeInformation.Symbols), func(contractSymbol binanceContractSymbolWire) bool {
		return contractSymbol.ContractType != binanceCryptoPerpetualContractType &&
			contractSymbol.ContractType != binanceTraditionalPerpetualContractType
	})
	slices.SortStableFunc(contractSymbols, func(left, right binanceContractSymbolWire) int {
		return cmp.Compare(right.OnboardDate, left.OnboardDate)
	})

	informationItems := []vo.InformationItemVo{}
	for _, contractSymbol := range contractSymbols[:min(itemLimit, len(contractSymbols))] {
		onboardedAt := time.UnixMilli(contractSymbol.OnboardDate).UTC()
		informationItems = append(informationItems, vo.InformationItemVo{
			SourceName:          binancePerpetualContractInformationSourceProxy.SourceName(),
			ExternalIdentifier:  contractSymbol.Symbol,
			Title:               contractSymbol.Symbol + " " + contractSymbol.ContractType + " contract onboarded",
			Link:                "https://www.binance.com/en/futures/" + contractSymbol.Symbol,
			PublishedAt:         &onboardedAt,
			DeclaredCoinSymbols: []string{contractSymbol.BaseAsset},
			IsTraditionalAsset:  contractSymbol.ContractType == binanceTraditionalPerpetualContractType,
		})
	}

	return informationItems, nil
}
