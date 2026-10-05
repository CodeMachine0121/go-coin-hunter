package vo

// CoinIdentityVo is how a candidate is looked up: by symbol, or exactly by the contract an on-chain source declared.
type CoinIdentityVo struct {
	CoinSymbol              string
	DeclaredContractAddress *TokenAddressVo
}
