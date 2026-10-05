package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_coin_market_data_proxy.go -destination=mocks/mock_i_coin_market_data_proxy.go -package=mocks

// ICoinMarketDataProxy is one free market data source; sources are injected as a list in priority order.
type ICoinMarketDataProxy interface {
	SourceName() string
	// FindCoinMarketData answers per coin symbol for the coins it knows; unknown coins are simply absent.
	FindCoinMarketData(executionContext context.Context, coinIdentities []vo.CoinIdentityVo) (map[string]vo.CoinMarketDataVo, error)
}
