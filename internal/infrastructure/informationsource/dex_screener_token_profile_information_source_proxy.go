package informationsource

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// dexScreenerAddressesPerLookup is DEX Screener's limit on token addresses per lookup request.
const dexScreenerAddressesPerLookup = 30

// DexScreenerTokenProfileInformationSourceProxy reads DEX Screener's newest token profiles; the profiles carry no
// ticker, so the tokens are looked up a second time to learn their symbol and when their first pair was created.
type DexScreenerTokenProfileInformationSourceProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewDexScreenerTokenProfileInformationSourceProxy(httpClient *http.Client, baseUrl string) *DexScreenerTokenProfileInformationSourceProxy {
	return &DexScreenerTokenProfileInformationSourceProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (dexScreenerTokenProfileInformationSourceProxy *DexScreenerTokenProfileInformationSourceProxy) SourceName() string {
	return "dexScreenerTokenProfile"
}

func (dexScreenerTokenProfileInformationSourceProxy *DexScreenerTokenProfileInformationSourceProxy) FetchInformationItems(
	executionContext context.Context, itemLimit int,
) ([]vo.InformationItemVo, error) {
	tokenProfiles, profilesError := getJson[[]dexScreenerTokenProfileWire](executionContext,
		dexScreenerTokenProfileInformationSourceProxy.httpClient,
		dexScreenerTokenProfileInformationSourceProxy.baseUrl+"/token-profiles/latest/v1")
	if profilesError != nil {
		return nil, profilesError
	}
	tokenProfiles = tokenProfiles[:min(itemLimit, len(tokenProfiles))]

	tokenAddressesByChain := map[string][]string{}
	chainOrder := []string{}
	for _, tokenProfile := range tokenProfiles {
		if _, known := tokenAddressesByChain[tokenProfile.ChainID]; !known {
			chainOrder = append(chainOrder, tokenProfile.ChainID)
		}
		tokenAddressesByChain[tokenProfile.ChainID] = append(tokenAddressesByChain[tokenProfile.ChainID], tokenProfile.TokenAddress)
	}

	// Keyed by chain and address; a token trades in several pairs, and its earliest pair is when it appeared.
	earliestPairByToken := map[string]dexScreenerPairWire{}
	for _, chainID := range chainOrder {
		tokenAddresses := tokenAddressesByChain[chainID]
		for batchStart := 0; batchStart < len(tokenAddresses); batchStart += dexScreenerAddressesPerLookup {
			batch := tokenAddresses[batchStart:min(batchStart+dexScreenerAddressesPerLookup, len(tokenAddresses))]
			pairs, pairsError := getJson[[]dexScreenerPairWire](executionContext,
				dexScreenerTokenProfileInformationSourceProxy.httpClient,
				fmt.Sprintf("%s/tokens/v1/%s/%s", dexScreenerTokenProfileInformationSourceProxy.baseUrl, chainID, strings.Join(batch, ",")))
			if pairsError != nil {
				return nil, pairsError
			}
			for _, pair := range pairs {
				tokenKey := chainID + ":" + strings.ToLower(pair.BaseToken.Address)
				earliestPair, known := earliestPairByToken[tokenKey]
				if !known || (pair.PairCreatedAt > 0 && (earliestPair.PairCreatedAt == 0 || pair.PairCreatedAt < earliestPair.PairCreatedAt)) {
					earliestPairByToken[tokenKey] = pair
				}
			}
		}
	}

	informationItems := []vo.InformationItemVo{}
	for _, tokenProfile := range tokenProfiles {
		tokenKey := tokenProfile.ChainID + ":" + strings.ToLower(tokenProfile.TokenAddress)
		earliestPair, known := earliestPairByToken[tokenKey]
		informationItem := vo.InformationItemVo{
			SourceName:         dexScreenerTokenProfileInformationSourceProxy.SourceName(),
			ExternalIdentifier: tokenKey,
			Title:              "New token profile on " + tokenProfile.ChainID,
			Link:               tokenProfile.Url,
		}
		if known {
			informationItem.Title = earliestPair.BaseToken.Name + " (" + earliestPair.BaseToken.Symbol + ") profiled on " + tokenProfile.ChainID
			informationItem.DeclaredCoinSymbols = []string{earliestPair.BaseToken.Symbol}
			if earliestPair.PairCreatedAt > 0 {
				pairCreatedAt := time.UnixMilli(earliestPair.PairCreatedAt).UTC()
				informationItem.PublishedAt = &pairCreatedAt
			}
		}
		informationItems = append(informationItems, informationItem)
	}

	return informationItems, nil
}
