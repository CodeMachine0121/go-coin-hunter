package informationsource

import (
	"context"
	"fmt"
	"net/http"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// CoinGeckoTrendingInformationSourceProxy reads CoinGecko's free trending list; it carries no publish time.
type CoinGeckoTrendingInformationSourceProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewCoinGeckoTrendingInformationSourceProxy(httpClient *http.Client, baseUrl string) *CoinGeckoTrendingInformationSourceProxy {
	return &CoinGeckoTrendingInformationSourceProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (coinGeckoTrendingInformationSourceProxy *CoinGeckoTrendingInformationSourceProxy) SourceName() string {
	return "coinGeckoTrending"
}

func (coinGeckoTrendingInformationSourceProxy *CoinGeckoTrendingInformationSourceProxy) FetchInformationItems(
	executionContext context.Context, itemLimit int,
) ([]vo.InformationItemVo, error) {
	trending, fetchError := getJson[coinGeckoTrendingWire](executionContext,
		coinGeckoTrendingInformationSourceProxy.httpClient,
		coinGeckoTrendingInformationSourceProxy.baseUrl+"/api/v3/search/trending")
	if fetchError != nil {
		return nil, fetchError
	}
	if trending.Coins == nil {
		return nil, fmt.Errorf("coingecko trending answered without a coin list")
	}

	informationItems := []vo.InformationItemVo{}
	for _, trendingCoin := range trending.Coins[:min(itemLimit, len(trending.Coins))] {
		informationItems = append(informationItems, vo.InformationItemVo{
			SourceName:          coinGeckoTrendingInformationSourceProxy.SourceName(),
			ExternalIdentifier:  trendingCoin.Item.ID,
			Title:               trendingCoin.Item.Name + " is trending on CoinGecko",
			Link:                "https://www.coingecko.com/en/coins/" + trendingCoin.Item.ID,
			DeclaredCoinSymbols: []string{trendingCoin.Item.Symbol},
		})
	}

	return informationItems, nil
}
