package marketdata

import (
	"context"
	"fmt"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// goPlusEvmChainIDs are GoPlus's numeric ids for the supported EVM chains; Solana has its own endpoint.
var goPlusEvmChainIDs = map[string]string{
	vo.ChainEthereum: "1", vo.ChainBsc: "56", vo.ChainBase: "8453", vo.ChainArbitrum: "42161", vo.ChainPolygon: "137",
}

const goPlusZeroAddress = "0x0000000000000000000000000000000000000000"

// GoPlusTokenSecurityProxy reads GoPlus's free token security endpoint, one contract per request, spaced so a round
// stays inside the free request rate.
type GoPlusTokenSecurityProxy struct {
	httpClient      *http.Client
	baseUrl         string
	requestInterval time.Duration
	pacing          sync.Mutex
	lastRequestAt   time.Time
}

func NewGoPlusTokenSecurityProxy(httpClient *http.Client, baseUrl string, requestInterval time.Duration) *GoPlusTokenSecurityProxy {
	return &GoPlusTokenSecurityProxy{httpClient: httpClient, baseUrl: baseUrl, requestInterval: requestInterval}
}

func (goPlusTokenSecurityProxy *GoPlusTokenSecurityProxy) SourceName() string {
	return "goPlusTokenSecurity"
}

func (goPlusTokenSecurityProxy *GoPlusTokenSecurityProxy) SupportsChain(chainID string) bool {
	_, evmSupported := goPlusEvmChainIDs[chainID]
	return evmSupported || chainID == vo.ChainSolana
}

// FindTokenSecurity treats mint and freeze powers as a risk only while someone still controls an EVM contract.
func (goPlusTokenSecurityProxy *GoPlusTokenSecurityProxy) FindTokenSecurity(
	executionContext context.Context, tokenAddress vo.TokenAddressVo,
) (vo.TokenSecurityVo, bool, error) {
	goPlusTokenSecurityProxy.waitForTurn(executionContext)

	if tokenAddress.ChainID == vo.ChainSolana {
		response, fetchError := utilities.GetJson[goPlusResponseWire[goPlusSolanaTokenSecurityWire]](executionContext, goPlusTokenSecurityProxy.httpClient,
			goPlusTokenSecurityProxy.baseUrl+"/api/v1/solana/token_security?contract_addresses="+url.QueryEscape(tokenAddress.Address))
		if fetchError != nil {
			return vo.TokenSecurityVo{}, false, fetchError
		}
		if response.Code != 1 {
			return vo.TokenSecurityVo{}, false, fmt.Errorf("goplus answered %d: %s", response.Code, response.Message)
		}
		finding, found := response.Result[tokenAddress.Address]
		if !found {
			return vo.TokenSecurityVo{}, false, nil
		}

		return vo.TokenSecurityVo{
			CannotSell:       finding.NonTransferable == "1",
			IsMintable:       finding.Mintable.Status == "1",
			CanFreezeHolders: finding.Freezable.Status == "1",
		}, true, nil
	}

	goPlusChainID, supported := goPlusEvmChainIDs[tokenAddress.ChainID]
	if !supported {
		return vo.TokenSecurityVo{}, false, nil
	}
	response, fetchError := utilities.GetJson[goPlusResponseWire[goPlusEvmTokenSecurityWire]](executionContext, goPlusTokenSecurityProxy.httpClient,
		fmt.Sprintf("%s/api/v1/token_security/%s?contract_addresses=%s", goPlusTokenSecurityProxy.baseUrl, goPlusChainID, url.QueryEscape(tokenAddress.Address)))
	if fetchError != nil {
		return vo.TokenSecurityVo{}, false, fetchError
	}
	if response.Code != 1 {
		return vo.TokenSecurityVo{}, false, fmt.Errorf("goplus answered %d: %s", response.Code, response.Message)
	}
	finding, found := response.Result[strings.ToLower(tokenAddress.Address)]
	if !found {
		return vo.TokenSecurityVo{}, false, nil
	}

	ownerAddress := strings.ToLower(strings.TrimSpace(finding.OwnerAddress))
	contractControlled := (ownerAddress != "" && ownerAddress != goPlusZeroAddress) ||
		finding.HiddenOwner == "1" || finding.CanTakeBackOwnership == "1"
	tokenSecurity := vo.TokenSecurityVo{
		IsHoneypot:       finding.IsHoneypot == "1",
		CannotSell:       finding.CannotSellAll == "1",
		IsMintable:       contractControlled && finding.IsMintable == "1",
		CanFreezeHolders: contractControlled && (finding.TransferPausable == "1" || finding.IsBlacklisted == "1"),
	}
	if buyTaxRate, parseError := decimal.NewFromString(finding.BuyTax); parseError == nil {
		tokenSecurity.BuyTaxRate = &buyTaxRate
	}
	if sellTaxRate, parseError := decimal.NewFromString(finding.SellTax); parseError == nil {
		tokenSecurity.SellTaxRate = &sellTaxRate
	}

	return tokenSecurity, true, nil
}

// waitForTurn spaces requests by the interval; holding the lock across the wait keeps concurrent callers in line.
func (goPlusTokenSecurityProxy *GoPlusTokenSecurityProxy) waitForTurn(executionContext context.Context) {
	goPlusTokenSecurityProxy.pacing.Lock()
	defer goPlusTokenSecurityProxy.pacing.Unlock()

	if wait := goPlusTokenSecurityProxy.requestInterval - time.Since(goPlusTokenSecurityProxy.lastRequestAt); wait > 0 {
		select {
		case <-time.After(wait):
		case <-executionContext.Done():
		}
	}
	goPlusTokenSecurityProxy.lastRequestAt = time.Now()
}
