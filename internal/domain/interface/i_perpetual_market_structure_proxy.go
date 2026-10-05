package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_perpetual_market_structure_proxy.go -destination=mocks/mock_i_perpetual_market_structure_proxy.go -package=mocks

// IPerpetualMarketStructureProxy is one exchange's view of a coin's USDT perpetual; exchanges are injected in priority order.
type IPerpetualMarketStructureProxy interface {
	// FindMarketStructure returns found=false when the exchange has no USDT perpetual for the coin.
	FindMarketStructure(executionContext context.Context, coinSymbol string) (marketStructure vo.PerpetualMarketStructureVo, found bool, findError error)
}
