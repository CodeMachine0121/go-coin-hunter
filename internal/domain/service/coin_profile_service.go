package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// securityCheckChainPriority is the order a coin's listed contracts are tried in when it was not found on-chain first.
var securityCheckChainPriority = []string{vo.ChainEthereum, vo.ChainBsc, vo.ChainBase, vo.ChainArbitrum, vo.ChainPolygon, vo.ChainSolana}

const securityNotQueriedWithoutPerpetualReason = "未上永續合約，未查詢安全資料"

// CoinProfileService is the only place filtering reaches outside: it gathers every candidate's coin profile in one pass.
type CoinProfileService struct {
	coinIntelligenceRepository      domaininterface.ICoinIntelligenceRepository
	coinMarketDataProxies           []domaininterface.ICoinMarketDataProxy
	tokenSecurityProxy              domaininterface.ITokenSecurityProxy
	perpetualContractListingProxies []domaininterface.IPerpetualContractListingProxy
	tokenUnlockScheduleProxy        domaininterface.ITokenUnlockScheduleProxy
	sourceRequestTimeout            time.Duration
}

func NewCoinProfileService(
	coinIntelligenceRepository domaininterface.ICoinIntelligenceRepository,
	coinMarketDataProxies []domaininterface.ICoinMarketDataProxy,
	tokenSecurityProxy domaininterface.ITokenSecurityProxy,
	perpetualContractListingProxies []domaininterface.IPerpetualContractListingProxy,
	tokenUnlockScheduleProxy domaininterface.ITokenUnlockScheduleProxy,
	sourceRequestTimeout time.Duration,
) *CoinProfileService {
	return &CoinProfileService{
		coinIntelligenceRepository:      coinIntelligenceRepository,
		coinMarketDataProxies:           coinMarketDataProxies,
		tokenSecurityProxy:              tokenSecurityProxy,
		perpetualContractListingProxies: perpetualContractListingProxies,
		tokenUnlockScheduleProxy:        tokenUnlockScheduleProxy,
		sourceRequestTimeout:            sourceRequestTimeout,
	}
}

// AssembleCoinProfiles returns one profile per candidate, in candidate order. A coin a source does not know is left
// blank; a source failing as a whole is ErrCoinProfileSourceUnavailable, naming the source.
func (coinProfileService *CoinProfileService) AssembleCoinProfiles(
	executionContext context.Context, coinCandidates []entities.CoinCandidate, evaluatedAt time.Time,
) ([]vo.CoinProfileVo, error) {
	coinSymbols := make([]string, 0, len(coinCandidates))
	for _, coinCandidate := range coinCandidates {
		coinSymbols = append(coinSymbols, coinCandidate.CoinSymbol)
	}
	declaredContractAddresses, declaredError := coinProfileService.coinIntelligenceRepository.FindDeclaredContractAddresses(
		executionContext, coinSymbols)
	if declaredError != nil {
		return nil, fmt.Errorf("find declared contract addresses: %w", declaredError)
	}
	coinIdentities := make([]vo.CoinIdentityVo, 0, len(coinSymbols))
	for _, coinSymbol := range coinSymbols {
		coinIdentity := vo.CoinIdentityVo{CoinSymbol: coinSymbol}
		if declaredContractAddress, declared := declaredContractAddresses[coinSymbol]; declared {
			coinIdentity.DeclaredContractAddress = &declaredContractAddress
		}
		coinIdentities = append(coinIdentities, coinIdentity)
	}

	// Market data: the first source in priority order that knows a coin speaks for it.
	marketDataBySymbol := map[string]vo.CoinMarketDataVo{}
	for _, coinMarketDataProxy := range coinProfileService.coinMarketDataProxies {
		sourceContext, cancelSource := context.WithTimeout(executionContext, coinProfileService.sourceRequestTimeout)
		foundMarketData, marketDataError := coinMarketDataProxy.FindCoinMarketData(sourceContext, coinIdentities)
		cancelSource()
		if marketDataError != nil {
			return nil, fmt.Errorf("%w：%s（%v）", domains.ErrCoinProfileSourceUnavailable, coinMarketDataProxy.SourceName(), marketDataError)
		}
		for coinSymbol, marketData := range foundMarketData {
			if _, known := marketDataBySymbol[coinSymbol]; !known {
				marketDataBySymbol[coinSymbol] = marketData
			}
		}
	}

	// Perpetual listings: every exchange at once; any exchange missing would make "not listed" a guess.
	listedCoinSymbolsByExchange := make([]map[string]bool, len(coinProfileService.perpetualContractListingProxies))
	listingErrors := make([]error, len(coinProfileService.perpetualContractListingProxies))
	waitGroup := sync.WaitGroup{}
	for index, perpetualContractListingProxy := range coinProfileService.perpetualContractListingProxies {
		waitGroup.Go(func() {
			sourceContext, cancelSource := context.WithTimeout(executionContext, coinProfileService.sourceRequestTimeout)
			defer cancelSource()
			listedCoinSymbolsByExchange[index], listingErrors[index] = perpetualContractListingProxy.FindUsdtPerpetualCoinSymbols(sourceContext)
		})
	}
	waitGroup.Wait()
	for index, listingError := range listingErrors {
		if listingError != nil {
			return nil, fmt.Errorf("%w：%s 永續合約清單（%v）", domains.ErrCoinProfileSourceUnavailable,
				coinProfileService.perpetualContractListingProxies[index].ExchangeName(), listingError)
		}
	}

	coinProfiles := make([]vo.CoinProfileVo, 0, len(coinIdentities))
	coinUnlockLookups := []vo.CoinUnlockLookupVo{}
	for _, coinIdentity := range coinIdentities {
		coinProfile := vo.CoinProfileVo{CoinSymbol: coinIdentity.CoinSymbol, EvaluatedAt: evaluatedAt, PerpetualContractExchanges: []string{}}
		for index, listedCoinSymbols := range listedCoinSymbolsByExchange {
			if listedCoinSymbols[coinIdentity.CoinSymbol] {
				coinProfile.PerpetualContractExchanges = append(coinProfile.PerpetualContractExchanges,
					coinProfileService.perpetualContractListingProxies[index].ExchangeName())
			}
		}

		// The contract to check: the one found on-chain, else the first listed contract on a supported chain.
		candidateAddresses := []vo.TokenAddressVo{}
		if coinIdentity.DeclaredContractAddress != nil {
			candidateAddresses = append(candidateAddresses, *coinIdentity.DeclaredContractAddress)
		}
		if marketData, known := marketDataBySymbol[coinIdentity.CoinSymbol]; known {
			coinProfile.MarketData = &marketData
			for _, chainID := range securityCheckChainPriority {
				for _, contractAddress := range marketData.ContractAddresses {
					if contractAddress.ChainID == chainID {
						candidateAddresses = append(candidateAddresses, contractAddress)
					}
				}
			}
			if marketData.CoinGeckoID != "" || marketData.Name != "" {
				coinUnlockLookups = append(coinUnlockLookups, vo.CoinUnlockLookupVo{
					CoinSymbol: coinIdentity.CoinSymbol, CoinGeckoID: marketData.CoinGeckoID, Name: marketData.Name,
				})
			}
		}
		for _, candidateAddress := range candidateAddresses {
			if coinProfileService.tokenSecurityProxy.SupportsChain(candidateAddress.ChainID) {
				coinProfile.ContractAddress = &candidateAddress
				break
			}
		}
		coinProfiles = append(coinProfiles, coinProfile)
	}

	// Security: one contract per request on the free quota, so only for coins the trader could trade at all.
	for index := range coinProfiles {
		coinProfile := &coinProfiles[index]
		if coinProfile.ContractAddress == nil {
			continue
		}
		if len(coinProfile.PerpetualContractExchanges) == 0 {
			coinProfile.TokenSecurityNotQueriedReason = securityNotQueriedWithoutPerpetualReason
			continue
		}
		sourceContext, cancelSource := context.WithTimeout(executionContext, coinProfileService.sourceRequestTimeout)
		tokenSecurity, found, securityError := coinProfileService.tokenSecurityProxy.FindTokenSecurity(sourceContext, *coinProfile.ContractAddress)
		cancelSource()
		if securityError != nil {
			return nil, fmt.Errorf("%w：%s（%v）", domains.ErrCoinProfileSourceUnavailable, coinProfileService.tokenSecurityProxy.SourceName(), securityError)
		}
		if found {
			coinProfile.TokenSecurity = &tokenSecurity
		}
	}

	if len(coinUnlockLookups) > 0 {
		sourceContext, cancelSource := context.WithTimeout(executionContext, coinProfileService.sourceRequestTimeout)
		unlockEventsBySymbol, unlockError := coinProfileService.tokenUnlockScheduleProxy.FindTokenUnlockEvents(sourceContext, coinUnlockLookups)
		cancelSource()
		if unlockError != nil {
			return nil, fmt.Errorf("%w：%s（%v）", domains.ErrCoinProfileSourceUnavailable, coinProfileService.tokenUnlockScheduleProxy.SourceName(), unlockError)
		}
		for index := range coinProfiles {
			if unlockEvents, covered := unlockEventsBySymbol[coinProfiles[index].CoinSymbol]; covered {
				coinProfiles[index].UnlockScheduleKnown = true
				coinProfiles[index].UnlockEvents = unlockEvents
			}
		}
	}

	return coinProfiles, nil
}
