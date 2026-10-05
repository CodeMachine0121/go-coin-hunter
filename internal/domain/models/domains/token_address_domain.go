package domains

import (
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// TokenAddressDomain knows how a chain compares contract addresses: EVM addresses ignore case, Solana's do not.
type TokenAddressDomain struct {
	tokenAddress vo.TokenAddressVo
}

func NewTokenAddressDomain(tokenAddress vo.TokenAddressVo) TokenAddressDomain {
	return TokenAddressDomain{tokenAddress: tokenAddress}
}

// ComparisonKey is equal for two addresses exactly when they name the same contract on the same chain.
func (tokenAddressDomain TokenAddressDomain) ComparisonKey() string {
	if tokenAddressDomain.tokenAddress.ChainID == vo.ChainSolana {
		return tokenAddressDomain.tokenAddress.ChainID + ":" + tokenAddressDomain.tokenAddress.Address
	}

	return tokenAddressDomain.tokenAddress.ChainID + ":" + strings.ToLower(tokenAddressDomain.tokenAddress.Address)
}
