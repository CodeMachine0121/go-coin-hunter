package marketdata

import (
	"context"
	"fmt"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"net/http"
	"net/url"
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// coinGeckoMarketsPageSize is the most coins one markets request returns.
const coinGeckoMarketsPageSize = 250

// coinGeckoChainIDs maps CoinGecko platform names onto the chain ids the domain uses.
var coinGeckoChainIDs = map[string]string{
	"ethereum": vo.ChainEthereum, "binance-smart-chain": vo.ChainBsc, "base": vo.ChainBase,
	"arbitrum-one": vo.ChainArbitrum, "polygon-pos": vo.ChainPolygon, "solana": vo.ChainSolana,
}

// CoinGeckoCoinMarketDataProxy reads CoinGecko's free coin list and markets. A coin found on-chain is matched by its
// contract; otherwise by symbol, taking the largest market cap among coins sharing it.
type CoinGeckoCoinMarketDataProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewCoinGeckoCoinMarketDataProxy(httpClient *http.Client, baseUrl string) *CoinGeckoCoinMarketDataProxy {
	return &CoinGeckoCoinMarketDataProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (coinGeckoCoinMarketDataProxy *CoinGeckoCoinMarketDataProxy) SourceName() string {
	return "coinGeckoMarketData"
}

func (coinGeckoCoinMarketDataProxy *CoinGeckoCoinMarketDataProxy) FindCoinMarketData(
	executionContext context.Context, coinIdentities []vo.CoinIdentityVo,
) (map[string]vo.CoinMarketDataVo, error) {
	listedCoins, listError := utilities.GetJson[[]coinGeckoListedCoinWire](executionContext, coinGeckoCoinMarketDataProxy.httpClient,
		coinGeckoCoinMarketDataProxy.baseUrl+"/api/v3/coins/list?include_platform=true")
	if listError != nil {
		return nil, listError
	}

	coinIDByContract := map[string]string{}
	coinIDsBySymbol := map[string][]string{}
	platformsByCoinID := map[string]map[string]string{}
	for _, listedCoin := range listedCoins {
		upperSymbol := strings.ToUpper(listedCoin.Symbol)
		coinIDsBySymbol[upperSymbol] = append(coinIDsBySymbol[upperSymbol], listedCoin.ID)
		platformsByCoinID[listedCoin.ID] = listedCoin.Platforms
		for platformName, contractAddress := range listedCoin.Platforms {
			if chainID, supported := coinGeckoChainIDs[platformName]; supported && contractAddress != "" {
				coinIDByContract[domains.NewTokenAddressDomain(vo.TokenAddressVo{ChainID: chainID, Address: contractAddress}).ComparisonKey()] = listedCoin.ID
			}
		}
	}

	coinIDsToPrice := []string{}
	possibleCoinIDsBySymbol := map[string][]string{}
	for _, coinIdentity := range coinIdentities {
		// A coin found on-chain is that contract and nothing else: an unknown contract is never stood in for by its symbol.
		possibleCoinIDs := coinIDsBySymbol[coinIdentity.CoinSymbol]
		if coinIdentity.DeclaredContractAddress != nil {
			possibleCoinIDs = []string{}
			if coinID, matched := coinIDByContract[domains.NewTokenAddressDomain(*coinIdentity.DeclaredContractAddress).ComparisonKey()]; matched {
				possibleCoinIDs = []string{coinID}
			}
		}
		possibleCoinIDsBySymbol[coinIdentity.CoinSymbol] = possibleCoinIDs
		coinIDsToPrice = append(coinIDsToPrice, possibleCoinIDs...)
	}

	marketsByCoinID := map[string]coinGeckoMarketWire{}
	for batchStart := 0; batchStart < len(coinIDsToPrice); batchStart += coinGeckoMarketsPageSize {
		batch := coinIDsToPrice[batchStart:min(batchStart+coinGeckoMarketsPageSize, len(coinIDsToPrice))]
		markets, marketsError := utilities.GetJson[[]coinGeckoMarketWire](executionContext, coinGeckoCoinMarketDataProxy.httpClient,
			fmt.Sprintf("%s/api/v3/coins/markets?vs_currency=usd&per_page=%d&ids=%s",
				coinGeckoCoinMarketDataProxy.baseUrl, coinGeckoMarketsPageSize, url.QueryEscape(strings.Join(batch, ","))))
		if marketsError != nil {
			return nil, marketsError
		}
		for _, market := range markets {
			marketsByCoinID[market.ID] = market
		}
	}

	marketDataBySymbol := map[string]vo.CoinMarketDataVo{}
	for coinSymbol, possibleCoinIDs := range possibleCoinIDsBySymbol {
		chosenMarket, chosen := coinGeckoMarketWire{}, false
		chosenMarketCap := decimal.NewFromInt(-1)
		for _, coinID := range possibleCoinIDs {
			market, priced := marketsByCoinID[coinID]
			if !priced {
				continue
			}
			marketCap := decimal.Zero
			if market.MarketCap != nil {
				marketCap = market.MarketCap.value
			}
			if marketCap.GreaterThan(chosenMarketCap) {
				chosenMarket, chosen, chosenMarketCap = market, true, marketCap
			}
		}
		if !chosen {
			continue
		}

		contractAddresses := []vo.TokenAddressVo{}
		for platformName, contractAddress := range platformsByCoinID[chosenMarket.ID] {
			if chainID, supported := coinGeckoChainIDs[platformName]; supported && contractAddress != "" {
				contractAddresses = append(contractAddresses, vo.TokenAddressVo{ChainID: chainID, Address: contractAddress})
			}
		}
		marketDataBySymbol[coinSymbol] = vo.CoinMarketDataVo{
			SourceName:               coinGeckoCoinMarketDataProxy.SourceName(),
			CoinGeckoID:              chosenMarket.ID,
			Name:                     chosenMarket.Name,
			FullyDilutedValuationUsd: chosenMarket.FullyDilutedValuation.decimalOrNil(),
			DailyVolumeUsd:           chosenMarket.TotalVolume.decimalOrNil(),
			CirculatingSupply:        chosenMarket.CirculatingSupply.decimalOrNil(),
			TotalSupply:              chosenMarket.TotalSupply.decimalOrNil(),
			MaxSupply:                chosenMarket.MaxSupply.decimalOrNil(),
			ContractAddresses:        contractAddresses,
		}
	}

	return marketDataBySymbol, nil
}
