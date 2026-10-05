package marketdata_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/marketdata"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveRoutes answers by path plus raw query when the route names one, else by path alone.
func serveRoutes(t *testing.T, routes map[string]string) string {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, known := routes[request.URL.Path+"?"+request.URL.RawQuery]
		if !known {
			body, known = routes[request.URL.Path]
		}
		if !known {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

func amount(text string) *decimal.Decimal {
	value := decimal.RequireFromString(text)
	return &value
}

func TestCoinGeckoMatchesByContractFirstThenLargestMarketCap(t *testing.T) {
	baseUrl := serveRoutes(t, map[string]string{
		"/api/v3/coins/list": `[
			{"id":"pudgy-penguins","symbol":"pengu","name":"Pudgy Penguins","platforms":{"solana":"2zMM","binance-smart-chain":"0xPENGU","unknown-chain":"x"}},
			{"id":"pengu-copy","symbol":"pengu","name":"Copy","platforms":{}},
			{"id":"cryptotwitter","symbol":"ct","name":"CryptoTwitter","platforms":{"ethereum":"0xCTWRONG"}},
			{"id":"cotton","symbol":"ct","name":"Cotton","platforms":{"ethereum":"0xC0770N"}},
			{"id":"nothing","symbol":"none","name":"Unpriced","platforms":{}},
			{"id":"pengu2-impostor","symbol":"pengu2","name":"Impostor","platforms":{}}]`,
		"/api/v3/coins/markets": `[
			{"id":"pudgy-penguins","name":"Pudgy Penguins","market_cap":613916781,"fully_diluted_valuation":868117026,"total_volume":"172066967.5","circulating_supply":62860396090,"total_supply":76722796386.4517,"max_supply":88888888888},
			{"id":"pengu-copy","name":"Copy","market_cap":null,"fully_diluted_valuation":null,"total_volume":5,"circulating_supply":null,"total_supply":null,"max_supply":null},
			{"id":"cryptotwitter","name":"CryptoTwitter","market_cap":33696},
			{"id":"cotton","name":"Cotton","market_cap":12,"max_supply":null},
			{"id":"pengu2-impostor","name":"Impostor","market_cap":999999999}]`,
	})
	proxy := marketdata.NewCoinGeckoCoinMarketDataProxy(http.DefaultClient, baseUrl)

	marketData, findError := proxy.FindCoinMarketData(context.Background(), []vo.CoinIdentityVo{
		{CoinSymbol: "PENGU"},
		{CoinSymbol: "CT", DeclaredContractAddress: &vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xc0770n"}},
		{CoinSymbol: "PENGU2", DeclaredContractAddress: &vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "UnknownToCoinGecko"}},
		{CoinSymbol: "NONE"},
		{CoinSymbol: "MISSING"},
	})

	require.NoError(t, findError)
	assert.Equal(t, "coinGeckoMarketData", proxy.SourceName())
	pengu := marketData["PENGU"]
	assert.Equal(t, "pudgy-penguins", pengu.CoinGeckoID)
	assert.Equal(t, "868117026", pengu.FullyDilutedValuationUsd.String())
	assert.Equal(t, "172066967.5", pengu.DailyVolumeUsd.String())
	assert.Equal(t, "88888888888", pengu.MaxSupply.String())
	assert.ElementsMatch(t, []vo.TokenAddressVo{{ChainID: vo.ChainSolana, Address: "2zMM"}, {ChainID: vo.ChainBsc, Address: "0xPENGU"}}, pengu.ContractAddresses)
	assert.Equal(t, "cotton", marketData["CT"].CoinGeckoID)
	assert.Nil(t, marketData["CT"].MaxSupply)
	assert.NotContains(t, marketData, "NONE")
	assert.NotContains(t, marketData, "PENGU2")
	assert.NotContains(t, marketData, "MISSING")
}

func TestDexScreenerAnswersOnlyForDeclaredContracts(t *testing.T) {
	baseUrl := serveRoutes(t, map[string]string{
		"/tokens/v1/solana/DouuMint": `[
			{"baseToken":{"address":"DouuMint","name":"Douu"},"fdv":3000000,"liquidity":{"usd":1000},"volume":{"h24":200}},
			{"baseToken":{"address":"DouuMint","name":"Douu"},"fdv":3100000,"liquidity":{"usd":5000},"volume":{"h24":300}},
			{"baseToken":{"address":"DouuMint","name":"Douu"},"fdv":9,"liquidity":null,"volume":null},
			{"baseToken":{"address":"Unrequested","name":"X"},"fdv":1}]`,
	})
	proxy := marketdata.NewDexScreenerCoinMarketDataProxy(http.DefaultClient, baseUrl)

	marketData, findError := proxy.FindCoinMarketData(context.Background(), []vo.CoinIdentityVo{
		{CoinSymbol: "DOUU", DeclaredContractAddress: &vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "DouuMint"}},
		{CoinSymbol: "SUI"},
	})

	require.NoError(t, findError)
	assert.Equal(t, "dexScreenerMarketData", proxy.SourceName())
	assert.Equal(t, map[string]vo.CoinMarketDataVo{"DOUU": {
		SourceName: "dexScreenerMarketData", Name: "Douu", FullyDilutedValuationUsd: amount("3100000"), DailyVolumeUsd: amount("500"),
		ContractAddresses: []vo.TokenAddressVo{{ChainID: vo.ChainSolana, Address: "DouuMint"}},
	}}, marketData)
}

func TestGoPlusReadsEvmAndSolanaFindings(t *testing.T) {
	baseUrl := serveRoutes(t, map[string]string{
		"/api/v1/token_security/1?contract_addresses=0xRenounced": `{"code":1,"result":{"0xrenounced":{"is_honeypot":"0","cannot_sell_all":"0","buy_tax":"0","sell_tax":"0.105",
			"is_mintable":"1","transfer_pausable":"1","is_blacklisted":"1","owner_address":"0x0000000000000000000000000000000000000000"}}}`,
		"/api/v1/token_security/56?contract_addresses=0xOwned": `{"code":1,"result":{"0xowned":{"is_honeypot":"1","cannot_sell_all":"1","buy_tax":"","sell_tax":"",
			"is_mintable":"1","transfer_pausable":"0","is_blacklisted":"1","owner_address":"0xc6cde7c39eb2f0f0095f41570af89efc2c1ea828"}}}`,
		"/api/v1/token_security/8453?contract_addresses=0xHidden":     `{"code":1,"result":{"0xhidden":{"is_mintable":"1","hidden_owner":"1"}}}`,
		"/api/v1/token_security/1?contract_addresses=0xUnknown":       `{"code":1,"result":{}}`,
		"/api/v1/token_security/1?contract_addresses=0xTakeBack":      `{"code":1,"result":{"0xtakeback":{"is_mintable":"1","owner_address":"0x0000000000000000000000000000000000000000","can_take_back_ownership":"1"}}}`,
		"/api/v1/token_security/1?contract_addresses=0xNoOwner":       `{"code":1,"result":{"0xnoowner":{"is_mintable":"1","is_blacklisted":"1","owner_address":""}}}`,
		"/api/v1/solana/token_security?contract_addresses=FreezeMint": `{"code":1,"result":{"FreezeMint":{"mintable":{"status":"0"},"freezable":{"status":"1"},"non_transferable":"1"}}}`,
		"/api/v1/token_security/1?contract_addresses=0xLimited":       `{"code":4029,"message":"too many requests","result":{}}`,
		"/api/v1/solana/token_security?contract_addresses=DouuMint":   `{"code":1,"result":{"DouuMint":{"mintable":{"status":"1"},"freezable":{"status":"0"},"non_transferable":"0"}}}`,
		"/api/v1/solana/token_security?contract_addresses=Gone":       `{"code":1,"result":{}}`,
		"/api/v1/solana/token_security?contract_addresses=Limited":    `{"code":4029,"message":"too many requests"}`,
	})
	proxy := marketdata.NewGoPlusTokenSecurityProxy(http.DefaultClient, baseUrl, 0)

	renounced, found, findError := proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xRenounced"})
	require.NoError(t, findError)
	assert.True(t, found)
	assert.Equal(t, vo.TokenSecurityVo{BuyTaxRate: amount("0"), SellTaxRate: amount("0.105")}, renounced)

	owned, _, _ := proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainBsc, Address: "0xOwned"})
	assert.Equal(t, vo.TokenSecurityVo{IsHoneypot: true, CannotSell: true, IsMintable: true, CanFreezeHolders: true}, owned)

	hidden, _, _ := proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainBase, Address: "0xHidden"})
	assert.True(t, hidden.IsMintable)

	takeBack, _, _ := proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xTakeBack"})
	assert.True(t, takeBack.IsMintable)

	noOwner, _, _ := proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xNoOwner"})
	assert.Equal(t, vo.TokenSecurityVo{}, noOwner)

	freezing, _, _ := proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "FreezeMint"})
	assert.Equal(t, vo.TokenSecurityVo{CannotSell: true, CanFreezeHolders: true}, freezing)

	_, found, findError = proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xUnknown"})
	require.NoError(t, findError)
	assert.False(t, found)

	_, _, findError = proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xLimited"})
	assert.ErrorContains(t, findError, "too many requests")

	solana, found, findError := proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "DouuMint"})
	require.NoError(t, findError)
	assert.True(t, found)
	assert.Equal(t, vo.TokenSecurityVo{IsMintable: true}, solana)

	_, found, _ = proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "Gone"})
	assert.False(t, found)
	_, _, findError = proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "Limited"})
	assert.Error(t, findError)
	_, found, findError = proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: "tron", Address: "T1"})
	assert.NoError(t, findError)
	assert.False(t, found)

	assert.Equal(t, "goPlusTokenSecurity", proxy.SourceName())
	assert.True(t, proxy.SupportsChain(vo.ChainPolygon))
	assert.True(t, proxy.SupportsChain(vo.ChainSolana))
	assert.False(t, proxy.SupportsChain("tron"))
}

func TestGoPlusSpacesItsRequests(t *testing.T) {
	baseUrl := serveRoutes(t, map[string]string{"/api/v1/token_security/1": `{"code":1,"result":{}}`})
	proxy := marketdata.NewGoPlusTokenSecurityProxy(http.DefaultClient, baseUrl, 100*time.Millisecond)
	startedAt := time.Now()

	for range 3 {
		_, _, findError := proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0x1"})
		require.NoError(t, findError)
	}

	assert.GreaterOrEqual(t, time.Since(startedAt), 200*time.Millisecond)
}

func TestPerpetualContractListingsKeepOnlyTradableCryptoUsdtPerpetuals(t *testing.T) {
	baseUrl := serveRoutes(t, map[string]string{
		"/fapi/v1/exchangeInfo": `{"symbols":[
			{"baseAsset":"ZORA","quoteAsset":"USDT","contractType":"PERPETUAL","status":"TRADING"},
			{"baseAsset":"NKE","quoteAsset":"USDT","contractType":"TRADIFI_PERPETUAL","status":"TRADING"},
			{"baseAsset":"BTC","quoteAsset":"USDT","contractType":"CURRENT_QUARTER","status":"TRADING"},
			{"baseAsset":"ETH","quoteAsset":"USDC","contractType":"PERPETUAL","status":"TRADING"},
			{"baseAsset":"OLD","quoteAsset":"USDT","contractType":"PERPETUAL","status":"SETTLING"}]}`,
		"/v5/market/instruments-info?category=linear&limit=1000&cursor=": `{"retCode":0,"result":{"list":[
			{"baseCoin":"0g","quoteCoin":"USDT","contractType":"LinearPerpetual","status":"Trading","symbolType":""},
			{"baseCoin":"AAPL","quoteCoin":"USDT","contractType":"LinearPerpetual","status":"Trading","symbolType":"stock"}],"nextPageCursor":"page2"}}`,
		"/v5/market/instruments-info?category=linear&limit=1000&cursor=page2": `{"retCode":0,"result":{"list":[
			{"baseCoin":"PUMP","quoteCoin":"USDT","contractType":"LinearPerpetual","status":"Trading","symbolType":"innovation"},
			{"baseCoin":"SOL","quoteCoin":"USDC","contractType":"LinearPerpetual","status":"Trading","symbolType":""},
			{"baseCoin":"BTC","quoteCoin":"USDT","contractType":"LinearFutures","status":"Trading","symbolType":""}],"nextPageCursor":""}}`,
		"/api/v5/public/instruments": `{"code":"0","data":[
			{"instFamily":"PENGU-USDT","settleCcy":"USDT","state":"live","instCategory":"1"},
			{"instFamily":"AAPL-USDT","settleCcy":"USDT","state":"live","instCategory":"3"},
			{"instFamily":"BTC-USD","settleCcy":"BTC","state":"live","instCategory":"1"},
			{"instFamily":"NEW-USDT","settleCcy":"USDT","state":"preopen","instCategory":"1"}]}`,
	})

	binance, binanceError := marketdata.NewBinancePerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
	bybit, bybitError := marketdata.NewBybitPerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
	okx, okxError := marketdata.NewOkxPerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())

	require.NoError(t, binanceError)
	require.NoError(t, bybitError)
	require.NoError(t, okxError)
	assert.Equal(t, map[string]bool{"ZORA": true}, binance)
	assert.Equal(t, map[string]bool{"0G": true, "PUMP": true}, bybit)
	assert.Equal(t, map[string]bool{"PENGU": true}, okx)
	assert.Equal(t, []string{"幣安", "Bybit", "OKX"}, []string{
		marketdata.NewBinancePerpetualContractListingProxy(nil, "").ExchangeName(),
		marketdata.NewBybitPerpetualContractListingProxy(nil, "").ExchangeName(),
		marketdata.NewOkxPerpetualContractListingProxy(nil, "").ExchangeName(),
	})
}

func TestDefiLlamaMatchesProtocolsAndSumsEachUnlock(t *testing.T) {
	baseUrl := serveRoutes(t, map[string]string{
		"/emissionsProtocolsList": `["grass","arbitrum","pump-fun","namesake"]`,
		"/emissions/grass": `{"name":"Grass","gecko_id":"grass","metadata":{"events":[
			{"timestamp":1793187432,"noOfTokens":[2710555.5]},{"timestamp":1793187442,"noOfTokens":[1000,"2000",null]}]}}`,
		"/emissions/pump-fun": `{"name":"Pump","gecko_id":"another-coin","metadata":{"events":[{"timestamp":1,"noOfTokens":[1]}]}}`,
		"/emissions/namesake": `{"name":"Namesake","gecko_id":"","metadata":{"events":[]}}`,
		"/emissions/arbitrum": `{"name":"Arbitrum","gecko_id":"arbitrum"}`,
	})
	proxy := marketdata.NewDefiLlamaTokenUnlockScheduleProxy(http.DefaultClient, baseUrl)

	unlockEvents, findError := proxy.FindTokenUnlockEvents(context.Background(), []vo.CoinUnlockLookupVo{
		{CoinSymbol: "GRASS", CoinGeckoID: "grass", Name: "Grass"},
		{CoinSymbol: "PUMP", CoinGeckoID: "pump-fun", Name: "Pump.fun"},
		{CoinSymbol: "NAME", CoinGeckoID: "namesake-token", Name: "Namesake"},
		{CoinSymbol: "ARB", CoinGeckoID: "arbitrum", Name: "Arbitrum"},
		{CoinSymbol: "ZORA", CoinGeckoID: "zora", Name: "Zora"},
	})

	require.NoError(t, findError)
	assert.Equal(t, "defiLlamaUnlockSchedule", proxy.SourceName())
	assert.Equal(t, map[string][]vo.TokenUnlockEventVo{
		"GRASS": {
			{UnlockAt: time.Unix(1793187432, 0).UTC(), Amount: decimal.RequireFromString("2710555.5")},
			{UnlockAt: time.Unix(1793187442, 0).UTC(), Amount: decimal.RequireFromString("3000")},
		},
		"NAME": {},
	}, unlockEvents)
}

func TestMarketDataProxiesFailOnBadAnswers(t *testing.T) {
	broken := func(t *testing.T, routes map[string]string) string { return serveRoutes(t, routes) }
	ethereumIdentity := []vo.CoinIdentityVo{{CoinSymbol: "CT", DeclaredContractAddress: &vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0x1"}}}
	symbolIdentity := []vo.CoinIdentityVo{{CoinSymbol: "CT"}}
	testCases := []struct {
		name  string
		fetch func(baseUrl string) error
		route map[string]string
	}{
		{name: "coingecko list unreachable", route: map[string]string{}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewCoinGeckoCoinMarketDataProxy(http.DefaultClient, baseUrl).FindCoinMarketData(context.Background(), ethereumIdentity)
			return fetchError
		}},
		{name: "coingecko markets unreachable", route: map[string]string{"/api/v3/coins/list": `[{"id":"ct","symbol":"ct","platforms":{}}]`}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewCoinGeckoCoinMarketDataProxy(http.DefaultClient, baseUrl).FindCoinMarketData(context.Background(), symbolIdentity)
			return fetchError
		}},
		{name: "coingecko a malformed figure", route: map[string]string{"/api/v3/coins/list": `[{"id":"ct","symbol":"ct","platforms":{}}]`,
			"/api/v3/coins/markets": `[{"id":"ct","market_cap":"lots"}]`}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewCoinGeckoCoinMarketDataProxy(http.DefaultClient, baseUrl).FindCoinMarketData(context.Background(), symbolIdentity)
			return fetchError
		}},
		{name: "dex screener unreachable", route: map[string]string{}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewDexScreenerCoinMarketDataProxy(http.DefaultClient, baseUrl).FindCoinMarketData(context.Background(), ethereumIdentity)
			return fetchError
		}},
		{name: "goplus evm unreachable", route: map[string]string{}, fetch: func(baseUrl string) error {
			_, _, fetchError := marketdata.NewGoPlusTokenSecurityProxy(http.DefaultClient, baseUrl, 0).FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0x1"})
			return fetchError
		}},
		{name: "goplus solana unreachable", route: map[string]string{}, fetch: func(baseUrl string) error {
			_, _, fetchError := marketdata.NewGoPlusTokenSecurityProxy(http.DefaultClient, baseUrl, 0).FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "S"})
			return fetchError
		}},
		{name: "binance unreachable", route: map[string]string{}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewBinancePerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "binance without a symbol list", route: map[string]string{"/fapi/v1/exchangeInfo": `{}`}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewBinancePerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "bybit unreachable", route: map[string]string{}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewBybitPerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "bybit without a list", route: map[string]string{"/v5/market/instruments-info": `{"retCode":0}`}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewBybitPerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "bybit refusing", route: map[string]string{"/v5/market/instruments-info": `{"retCode":10006,"retMsg":"rate limit","result":{"list":[]}}`}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewBybitPerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "okx unreachable", route: map[string]string{}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewOkxPerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "okx without a list", route: map[string]string{"/api/v5/public/instruments": `{"code":"0"}`}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewOkxPerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "defillama list unreachable", route: map[string]string{}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewDefiLlamaTokenUnlockScheduleProxy(http.DefaultClient, baseUrl).FindTokenUnlockEvents(context.Background(), nil)
			return fetchError
		}},
		{name: "an unparseable address", route: map[string]string{}, fetch: func(string) error {
			_, fetchError := marketdata.NewOkxPerpetualContractListingProxy(http.DefaultClient, "http://bad host").FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "a refused connection", route: map[string]string{}, fetch: func(string) error {
			_, fetchError := marketdata.NewOkxPerpetualContractListingProxy(http.DefaultClient, "http://127.0.0.1:1").FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
		{name: "a malformed body", route: map[string]string{"/api/v5/public/instruments": `not json`}, fetch: func(baseUrl string) error {
			_, fetchError := marketdata.NewOkxPerpetualContractListingProxy(http.DefaultClient, baseUrl).FindUsdtPerpetualCoinSymbols(context.Background())
			return fetchError
		}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Error(t, testCase.fetch(broken(t, testCase.route)))
		})
	}
}

func TestGoPlusStopsWaitingWhenTheContextEnds(t *testing.T) {
	proxy := marketdata.NewGoPlusTokenSecurityProxy(http.DefaultClient, serveRoutes(t, map[string]string{"/api/v1/token_security/1": `{"code":1,"result":{}}`}), time.Hour)
	_, _, _ = proxy.FindTokenSecurity(context.Background(), vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0x1"})
	expired, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, findError := proxy.FindTokenSecurity(expired, vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0x1"})

	assert.ErrorIs(t, findError, context.Canceled)
}

func TestDefiLlamaLeavesACoinUncoveredWhenItsDatasetCannotBeRead(t *testing.T) {
	proxy := marketdata.NewDefiLlamaTokenUnlockScheduleProxy(http.DefaultClient, serveRoutes(t, map[string]string{
		"/emissionsProtocolsList": `["grass","zora"]`,
		"/emissions/zora":         `{"name":"Zora","gecko_id":"zora","metadata":{"events":[]}}`,
	}))

	unlockEvents, findError := proxy.FindTokenUnlockEvents(context.Background(), []vo.CoinUnlockLookupVo{
		{CoinSymbol: "GRASS", CoinGeckoID: "grass", Name: "Grass"}, {CoinSymbol: "ZORA", CoinGeckoID: "zora", Name: "Zora"},
	})

	require.NoError(t, findError)
	assert.Equal(t, map[string][]vo.TokenUnlockEventVo{"ZORA": {}}, unlockEvents)
}

// countingServer answers every request with the body and records how many items each request asked for.
func countingServer(t *testing.T, body string, itemsOf func(request *http.Request) int) (string, *[]int) {
	itemCounts := []int{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if count := itemsOf(request); count > 0 {
			itemCounts = append(itemCounts, count)
		}
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server.URL, &itemCounts
}

func TestMarketDataLookupsStayInsideBatchLimits(t *testing.T) {
	t.Run("coingecko prices at most 250 coins per request", func(t *testing.T) {
		coinList := "["
		coinIdentities := []vo.CoinIdentityVo{}
		for index := range 251 {
			if index > 0 {
				coinList += ","
			}
			coinList += fmt.Sprintf(`{"id":"coin-%d","symbol":"c%d","platforms":{}}`, index, index)
			coinIdentities = append(coinIdentities, vo.CoinIdentityVo{CoinSymbol: fmt.Sprintf("C%d", index)})
		}
		coinList += "]"
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/api/v3/coins/list" {
				_, _ = writer.Write([]byte(coinList))
				return
			}
			_, _ = writer.Write([]byte(`[]`))
		}))
		t.Cleanup(server.Close)
		requestedBatches := []int{}
		recordingClient := &http.Client{Transport: roundTripRecorder(func(request *http.Request) {
			if request.URL.Path == "/api/v3/coins/markets" {
				requestedBatches = append(requestedBatches, len(strings.Split(request.URL.Query().Get("ids"), ",")))
			}
		})}

		_, findError := marketdata.NewCoinGeckoCoinMarketDataProxy(recordingClient, server.URL).FindCoinMarketData(context.Background(), coinIdentities)

		require.NoError(t, findError)
		assert.Equal(t, []int{250, 1}, requestedBatches)
	})

	t.Run("dex screener looks up at most 30 contracts per request", func(t *testing.T) {
		coinIdentities := []vo.CoinIdentityVo{}
		for index := range 31 {
			coinIdentities = append(coinIdentities, vo.CoinIdentityVo{CoinSymbol: fmt.Sprintf("T%d", index),
				DeclaredContractAddress: &vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: fmt.Sprintf("Mint%d", index)}})
		}
		baseUrl, itemCounts := countingServer(t, `[]`, func(request *http.Request) int {
			return len(strings.Split(strings.TrimPrefix(request.URL.Path, "/tokens/v1/solana/"), ","))
		})

		_, findError := marketdata.NewDexScreenerCoinMarketDataProxy(http.DefaultClient, baseUrl).FindCoinMarketData(context.Background(), coinIdentities)

		require.NoError(t, findError)
		assert.Equal(t, []int{30, 1}, *itemCounts)
	})
}

// roundTripRecorder sees each outgoing request before the default transport sends it.
type roundTripRecorder func(request *http.Request)

func (recorder roundTripRecorder) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder(request)
	return http.DefaultTransport.RoundTrip(request)
}
