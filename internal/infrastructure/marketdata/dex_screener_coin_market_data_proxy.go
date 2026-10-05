package marketdata

import (
	"context"
	"fmt"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"net/http"
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

const dexScreenerAddressesPerLookup = 30

// DexScreenerCoinMarketDataProxy answers for coins found on-chain only: valuation from the most liquid pair and
// volume summed over every pair. It knows no supply figures.
type DexScreenerCoinMarketDataProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewDexScreenerCoinMarketDataProxy(httpClient *http.Client, baseUrl string) *DexScreenerCoinMarketDataProxy {
	return &DexScreenerCoinMarketDataProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (dexScreenerCoinMarketDataProxy *DexScreenerCoinMarketDataProxy) SourceName() string {
	return "dexScreenerMarketData"
}

func (dexScreenerCoinMarketDataProxy *DexScreenerCoinMarketDataProxy) FindCoinMarketData(
	executionContext context.Context, coinIdentities []vo.CoinIdentityVo,
) (map[string]vo.CoinMarketDataVo, error) {
	coinSymbolByContract := map[string]string{}
	addressesByChain := map[string][]string{}
	chainOrder := []string{}
	for _, coinIdentity := range coinIdentities {
		if coinIdentity.DeclaredContractAddress == nil {
			continue
		}
		chainID := coinIdentity.DeclaredContractAddress.ChainID
		if _, known := addressesByChain[chainID]; !known {
			chainOrder = append(chainOrder, chainID)
		}
		addressesByChain[chainID] = append(addressesByChain[chainID], coinIdentity.DeclaredContractAddress.Address)
		coinSymbolByContract[domains.NewTokenAddressDomain(vo.TokenAddressVo{ChainID: chainID, Address: coinIdentity.DeclaredContractAddress.Address}).ComparisonKey()] = coinIdentity.CoinSymbol
	}

	marketDataBySymbol := map[string]vo.CoinMarketDataVo{}
	mostLiquidUsdBySymbol := map[string]decimal.Decimal{}
	for _, chainID := range chainOrder {
		addresses := addressesByChain[chainID]
		for batchStart := 0; batchStart < len(addresses); batchStart += dexScreenerAddressesPerLookup {
			batch := addresses[batchStart:min(batchStart+dexScreenerAddressesPerLookup, len(addresses))]
			pairs, pairsError := utilities.GetJson[[]dexScreenerPairWire](executionContext, dexScreenerCoinMarketDataProxy.httpClient,
				fmt.Sprintf("%s/tokens/v1/%s/%s", dexScreenerCoinMarketDataProxy.baseUrl, chainID, strings.Join(batch, ",")))
			if pairsError != nil {
				return nil, pairsError
			}
			for _, pair := range pairs {
				coinSymbol, requested := coinSymbolByContract[domains.NewTokenAddressDomain(vo.TokenAddressVo{ChainID: chainID, Address: pair.BaseToken.Address}).ComparisonKey()]
				if !requested {
					continue
				}
				marketData, seen := marketDataBySymbol[coinSymbol]
				if !seen {
					marketData = vo.CoinMarketDataVo{
						SourceName:        dexScreenerCoinMarketDataProxy.SourceName(),
						Name:              pair.BaseToken.Name,
						ContractAddresses: []vo.TokenAddressVo{{ChainID: chainID, Address: pair.BaseToken.Address}},
					}
					mostLiquidUsdBySymbol[coinSymbol] = decimal.NewFromInt(-1)
				}
				if pair.Volume != nil && pair.Volume.Hours24 != nil {
					dailyVolumeUsd := pair.Volume.Hours24.value
					if marketData.DailyVolumeUsd != nil {
						dailyVolumeUsd = dailyVolumeUsd.Add(*marketData.DailyVolumeUsd)
					}
					marketData.DailyVolumeUsd = &dailyVolumeUsd
				}
				liquidityUsd := decimal.Zero
				if pair.Liquidity != nil && pair.Liquidity.Usd != nil {
					liquidityUsd = pair.Liquidity.Usd.value
				}
				if liquidityUsd.GreaterThan(mostLiquidUsdBySymbol[coinSymbol]) {
					mostLiquidUsdBySymbol[coinSymbol] = liquidityUsd
					marketData.FullyDilutedValuationUsd = pair.FullyDilutedValuation.decimalOrNil()
				}
				marketDataBySymbol[coinSymbol] = marketData
			}
		}
	}

	return marketDataBySymbol, nil
}
