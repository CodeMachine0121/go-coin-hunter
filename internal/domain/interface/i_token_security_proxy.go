package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_token_security_proxy.go -destination=mocks/mock_i_token_security_proxy.go -package=mocks

type ITokenSecurityProxy interface {
	SourceName() string
	// SupportsChain tells whether the source can check contracts on that chain at all.
	SupportsChain(chainID string) bool
	// FindTokenSecurity returns found=false when the source has no findings for the contract.
	FindTokenSecurity(executionContext context.Context, tokenAddress vo.TokenAddressVo) (tokenSecurity vo.TokenSecurityVo, found bool, findError error)
}
