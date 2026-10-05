package _interface

import "context"

//go:generate go tool mockgen -source=i_perpetual_contract_listing_proxy.go -destination=mocks/mock_i_perpetual_contract_listing_proxy.go -package=mocks

// IPerpetualContractListingProxy is one exchange's list of tradable USDT perpetual contracts; exchanges are injected as a list.
type IPerpetualContractListingProxy interface {
	// ExchangeName is how the exchange is named in reasons, such as 幣安.
	ExchangeName() string
	// FindUsdtPerpetualCoinSymbols returns the upper-case base coin of every tradable, crypto-native USDT perpetual.
	FindUsdtPerpetualCoinSymbols(executionContext context.Context) (map[string]bool, error)
}
