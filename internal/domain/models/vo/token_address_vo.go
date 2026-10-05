package vo

// TokenAddressVo locates a token contract on one chain; ChainID uses the names in the Chain* constants where known.
type TokenAddressVo struct {
	ChainID string
	Address string
}

// The mainstream chains the filtering step can check; any other chain id is carried as the source spelled it.
const (
	ChainEthereum = "ethereum"
	ChainBsc      = "bsc"
	ChainBase     = "base"
	ChainArbitrum = "arbitrum"
	ChainPolygon  = "polygon"
	ChainSolana   = "solana"
)
