package marketdata

import (
	"context"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// nonSlugCharacters are replaced by hyphens when a coin name is turned into a DefiLlama protocol slug.
var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

// DefiLlamaTokenUnlockScheduleProxy reads DefiLlama's free public emissions datasets. A coin is matched to a protocol
// by its CoinGecko id or its slugged name, then confirmed by the protocol's own CoinGecko id or name.
type DefiLlamaTokenUnlockScheduleProxy struct {
	httpClient *http.Client
	baseUrl    string
}

func NewDefiLlamaTokenUnlockScheduleProxy(httpClient *http.Client, baseUrl string) *DefiLlamaTokenUnlockScheduleProxy {
	return &DefiLlamaTokenUnlockScheduleProxy{httpClient: httpClient, baseUrl: baseUrl}
}

func (defiLlamaTokenUnlockScheduleProxy *DefiLlamaTokenUnlockScheduleProxy) SourceName() string {
	return "defiLlamaUnlockSchedule"
}

func (defiLlamaTokenUnlockScheduleProxy *DefiLlamaTokenUnlockScheduleProxy) FindTokenUnlockEvents(
	executionContext context.Context, coinUnlockLookups []vo.CoinUnlockLookupVo,
) (map[string][]vo.TokenUnlockEventVo, error) {
	protocolSlugs, listError := utilities.GetJson[[]string](executionContext, defiLlamaTokenUnlockScheduleProxy.httpClient,
		defiLlamaTokenUnlockScheduleProxy.baseUrl+"/emissionsProtocolsList")
	if listError != nil {
		return nil, listError
	}
	knownSlugs := map[string]bool{}
	for _, protocolSlug := range protocolSlugs {
		knownSlugs[protocolSlug] = true
	}

	unlockEventsBySymbol := map[string][]vo.TokenUnlockEventVo{}
	for _, coinUnlockLookup := range coinUnlockLookups {
		nameSlug := strings.Trim(nonSlugCharacters.ReplaceAllString(strings.ToLower(coinUnlockLookup.Name), "-"), "-")
		for _, protocolSlug := range []string{coinUnlockLookup.CoinGeckoID, nameSlug} {
			if protocolSlug == "" || !knownSlugs[protocolSlug] {
				continue
			}
			emission, emissionError := utilities.GetJson[defiLlamaEmissionWire](executionContext, defiLlamaTokenUnlockScheduleProxy.httpClient,
				defiLlamaTokenUnlockScheduleProxy.baseUrl+"/emissions/"+url.PathEscape(protocolSlug))
			if emissionError != nil {
				return nil, emissionError
			}
			sameCoin := (emission.GeckoID != "" && emission.GeckoID == coinUnlockLookup.CoinGeckoID) ||
				(emission.GeckoID == "" && strings.EqualFold(emission.Name, coinUnlockLookup.Name))
			if !sameCoin || emission.Metadata == nil {
				continue
			}

			unlockEvents := []vo.TokenUnlockEventVo{}
			for _, event := range emission.Metadata.Events {
				amount := decimal.Zero
				for _, tokenCount := range event.NoOfTokens {
					if tokenCount != nil {
						amount = amount.Add(tokenCount.value)
					}
				}
				unlockEvents = append(unlockEvents, vo.TokenUnlockEventVo{UnlockAt: time.Unix(event.Timestamp, 0).UTC(), Amount: amount})
			}
			unlockEventsBySymbol[coinUnlockLookup.CoinSymbol] = unlockEvents
			break
		}
	}

	return unlockEventsBySymbol, nil
}
