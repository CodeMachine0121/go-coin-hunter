package informationsource_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/informationsource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveRoutes answers each request path with a fixed body, standing in for the real source.
func serveRoutes(t *testing.T, routes map[string]string) string {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, known := routes[request.URL.Path]
		if !known {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

func millis(milliseconds int64) *time.Time {
	publishedAt := time.UnixMilli(milliseconds).UTC()
	return &publishedAt
}

func TestInformationSourceProxiesNormalizeTheirSources(t *testing.T) {
	testCases := []struct {
		name      string
		routes    map[string]string
		newProxy  func(baseUrl string) domaininterface.IInformationSourceProxy
		itemLimit int
		want      []vo.InformationItemVo
	}{
		{
			name: "binance announcements",
			routes: map[string]string{"/bapi/composite/v1/public/cms/article/list/query": `{"code":"000000","data":{"catalogs":[{"articles":[
				{"code":"c1","title":"Binance Will List Cotton (CT)","releaseDate":1790836201040},
				{"code":"c2","title":"second","releaseDate":1790836201041}]}]}}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBinanceAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 1,
			want: []vo.InformationItemVo{{SourceName: "binanceAnnouncement", ExternalIdentifier: "c1", Title: "Binance Will List Cotton (CT)",
				Link: "https://www.binance.com/en/support/announcement/c1", PublishedAt: millis(1790836201040)}},
		},
		{
			name: "binance perpetual contracts, newest first, delivery contracts left out, tradfi marked",
			routes: map[string]string{"/fapi/v1/exchangeInfo": `{"symbols":[
				{"symbol":"BTCUSDT","baseAsset":"BTC","contractType":"PERPETUAL","onboardDate":1},
				{"symbol":"BTCUSDT_261226","baseAsset":"BTC","contractType":"CURRENT_QUARTER","onboardDate":9},
				{"symbol":"NKEUSDT","baseAsset":"NKE","contractType":"TRADIFI_PERPETUAL","onboardDate":2},
				{"symbol":"CTUSDT","baseAsset":"CT","contractType":"PERPETUAL","onboardDate":3}]}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBinancePerpetualContractInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 2,
			want: []vo.InformationItemVo{
				{SourceName: "binancePerpetualContract", ExternalIdentifier: "CTUSDT", Title: "CTUSDT PERPETUAL contract onboarded",
					Link: "https://www.binance.com/en/futures/CTUSDT", PublishedAt: millis(3), DeclaredCoinSymbols: []string{"CT"}},
				{SourceName: "binancePerpetualContract", ExternalIdentifier: "NKEUSDT", Title: "NKEUSDT TRADIFI_PERPETUAL contract onboarded",
					Link: "https://www.binance.com/en/futures/NKEUSDT", PublishedAt: millis(2), DeclaredCoinSymbols: []string{"NKE"}, IsTraditionalAsset: true},
			},
		},
		{
			name: "bybit announcements",
			routes: map[string]string{"/v5/announcements/index": `{"retCode":0,"result":{"list":[
				{"title":"New listing: CTUSDT Perpetual Contract","url":"https://bybit/a","dateTimestamp":1790846811000},
				{"title":"beyond the limit","url":"https://bybit/b","dateTimestamp":1790846811001}]}}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBybitAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 1,
			want: []vo.InformationItemVo{{SourceName: "bybitAnnouncement", ExternalIdentifier: "https://bybit/a",
				Title: "New listing: CTUSDT Perpetual Contract", Link: "https://bybit/a", PublishedAt: millis(1790846811000)}},
		},
		{
			name: "okx announcements",
			routes: map[string]string{"/api/v5/support/announcements": `{"code":"0","data":[{"details":[
				{"title":"OKX to list Cotton (CT)","url":"https://okx/a","pTime":"1790915409959"},
				{"title":"later","url":"https://okx/b","pTime":"1790915409960"}]}]}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewOkxAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 1,
			want: []vo.InformationItemVo{{SourceName: "okxAnnouncement", ExternalIdentifier: "https://okx/a",
				Title: "OKX to list Cotton (CT)", Link: "https://okx/a", PublishedAt: millis(1790915409959)}},
		},
		{
			name: "okx announcements fewer than the limit",
			routes: map[string]string{"/api/v5/support/announcements": `{"code":"0","data":[{"details":[
				{"title":"OKX to list Cotton (CT)","url":"https://okx/a","pTime":"1790915409959"}]}]}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewOkxAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 50,
			want: []vo.InformationItemVo{{SourceName: "okxAnnouncement", ExternalIdentifier: "https://okx/a",
				Title: "OKX to list Cotton (CT)", Link: "https://okx/a", PublishedAt: millis(1790915409959)}},
		},
		{
			name:   "coingecko trending, no publish time",
			routes: map[string]string{"/api/v3/search/trending": `{"coins":[{"item":{"id":"fetch-ai","symbol":"FET","name":"Fetch"}}]}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewCoinGeckoTrendingInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 50,
			want: []vo.InformationItemVo{{SourceName: "coinGeckoTrending", ExternalIdentifier: "fetch-ai", Title: "Fetch is trending on CoinGecko",
				Link: "https://www.coingecko.com/en/coins/fetch-ai", DeclaredCoinSymbols: []string{"FET"}}},
		},
		{
			name:   "coingecko trending cut at the limit",
			routes: map[string]string{"/api/v3/search/trending": `{"coins":[{"item":{"id":"a","symbol":"AAA","name":"A"}},{"item":{"id":"b","symbol":"BBB","name":"B"}}]}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewCoinGeckoTrendingInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 1,
			want: []vo.InformationItemVo{{SourceName: "coinGeckoTrending", ExternalIdentifier: "a", Title: "A is trending on CoinGecko",
				Link: "https://www.coingecko.com/en/coins/a", DeclaredCoinSymbols: []string{"AAA"}}},
		},
		{
			name: "dex screener profiles cut at the limit before looking up",
			routes: map[string]string{
				"/token-profiles/latest/v1": `[{"url":"https://dex/a","chainId":"solana","tokenAddress":"TokA"},{"url":"https://dex/b","chainId":"solana","tokenAddress":"TokB"}]`,
				"/tokens/v1/solana/TokA":    `[{"chainId":"solana","baseToken":{"address":"TokA","name":"Douu","symbol":"DOUU"},"pairCreatedAt":1000}]`,
			},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewDexScreenerTokenProfileInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 1,
			want: []vo.InformationItemVo{{SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "solana:toka", Title: "Douu (DOUU) profiled on solana",
				Link: "https://dex/a", PublishedAt: millis(1000), DeclaredCoinSymbols: []string{"DOUU"}}},
		},
		{
			name: "dex screener profiles, symbol and earliest pair looked up, unknown token keeps no coin",
			routes: map[string]string{
				"/token-profiles/latest/v1": `[{"url":"https://dex/a","chainId":"solana","tokenAddress":"TokA"},
					{"url":"https://dex/b","chainId":"solana","tokenAddress":"TokB"}]`,
				"/tokens/v1/solana/TokA,TokB": `[
					{"chainId":"solana","baseToken":{"address":"TokA","name":"Douu","symbol":"DOUU"},"pairCreatedAt":0},
					{"chainId":"solana","baseToken":{"address":"toka","name":"Douu","symbol":"DOUU"},"pairCreatedAt":2000},
					{"chainId":"solana","baseToken":{"address":"TokA","name":"Douu","symbol":"DOUU"},"pairCreatedAt":1000}]`,
			},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewDexScreenerTokenProfileInformationSourceProxy(http.DefaultClient, baseUrl)
			},
			itemLimit: 50,
			want: []vo.InformationItemVo{
				{SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "solana:toka", Title: "Douu (DOUU) profiled on solana",
					Link: "https://dex/a", PublishedAt: millis(1000), DeclaredCoinSymbols: []string{"DOUU"}},
				{SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "solana:tokb", Title: "New token profile on solana", Link: "https://dex/b"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			informationSourceProxy := testCase.newProxy(serveRoutes(t, testCase.routes))

			informationItems, fetchError := informationSourceProxy.FetchInformationItems(context.Background(), testCase.itemLimit)

			require.NoError(t, fetchError)
			assert.Equal(t, testCase.want, informationItems)
		})
	}
}

func TestInformationSourceProxiesFailOnBadAnswers(t *testing.T) {
	testCases := []struct {
		name     string
		routes   map[string]string
		newProxy func(baseUrl string) domaininterface.IInformationSourceProxy
	}{
		{name: "bybit answering without a return code", routes: map[string]string{"/v5/announcements/index": `{}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBybitAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "binance contracts answering without a symbol list", routes: map[string]string{"/fapi/v1/exchangeInfo": `{}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBinancePerpetualContractInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "coingecko answering without a coin list", routes: map[string]string{"/api/v3/search/trending": `{}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewCoinGeckoTrendingInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "binance announcements answering empty", routes: map[string]string{"/bapi/composite/v1/public/cms/article/list/query": `{}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBinanceAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "okx answering empty", routes: map[string]string{"/api/v5/support/announcements": `{}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewOkxAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "binance refusing", routes: map[string]string{"/bapi/composite/v1/public/cms/article/list/query": `{"code":"100001","data":{"catalogs":[]}}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBinanceAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "binance announcements unreachable", routes: map[string]string{},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBinanceAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "binance contracts unreachable", routes: map[string]string{},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBinancePerpetualContractInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "bybit refusing", routes: map[string]string{"/v5/announcements/index": `{"retCode":10001,"retMsg":"bad"}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBybitAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "bybit unreachable", routes: map[string]string{},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewBybitAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "okx refusing", routes: map[string]string{"/api/v5/support/announcements": `{"code":"50011","msg":"too many"}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewOkxAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "okx unreachable", routes: map[string]string{},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewOkxAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "okx unreadable publish time", routes: map[string]string{"/api/v5/support/announcements": `{"code":"0","data":[{"details":[{"title":"t","pTime":"soon"}]}]}`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewOkxAnnouncementInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "coingecko malformed", routes: map[string]string{"/api/v3/search/trending": `not json`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewCoinGeckoTrendingInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "dex screener profiles unreachable", routes: map[string]string{},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewDexScreenerTokenProfileInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "dex screener lookup unreachable", routes: map[string]string{"/token-profiles/latest/v1": `[{"chainId":"solana","tokenAddress":"TokA"}]`},
			newProxy: func(baseUrl string) domaininterface.IInformationSourceProxy {
				return informationsource.NewDexScreenerTokenProfileInformationSourceProxy(http.DefaultClient, baseUrl)
			}},
		{name: "an unparseable address", routes: map[string]string{},
			newProxy: func(string) domaininterface.IInformationSourceProxy {
				return informationsource.NewCoinGeckoTrendingInformationSourceProxy(http.DefaultClient, "http://bad host")
			}},
		{name: "a refused connection", routes: map[string]string{},
			newProxy: func(string) domaininterface.IInformationSourceProxy {
				return informationsource.NewCoinGeckoTrendingInformationSourceProxy(http.DefaultClient, "http://127.0.0.1:1")
			}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			informationSourceProxy := testCase.newProxy(serveRoutes(t, testCase.routes))

			_, fetchError := informationSourceProxy.FetchInformationItems(context.Background(), 50)

			assert.Error(t, fetchError)
		})
	}
}

func TestInformationSourceProxiesNameThemselves(t *testing.T) {
	assert.Equal(t, []string{"binanceAnnouncement", "binancePerpetualContract", "bybitAnnouncement", "okxAnnouncement",
		"coinGeckoTrending", "dexScreenerTokenProfile"}, []string{
		informationsource.NewBinanceAnnouncementInformationSourceProxy(nil, "").SourceName(),
		informationsource.NewBinancePerpetualContractInformationSourceProxy(nil, "").SourceName(),
		informationsource.NewBybitAnnouncementInformationSourceProxy(nil, "").SourceName(),
		informationsource.NewOkxAnnouncementInformationSourceProxy(nil, "").SourceName(),
		informationsource.NewCoinGeckoTrendingInformationSourceProxy(nil, "").SourceName(),
		informationsource.NewDexScreenerTokenProfileInformationSourceProxy(nil, "").SourceName(),
	})
}
