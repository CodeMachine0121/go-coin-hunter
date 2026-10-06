package service

import (
	"context"
	"sync"
	"time"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// PerpetualMarketStructureService answers "how does this coin's perpetual trade right now" from the exchanges in
// priority order: the first that lists the coin speaks for it, and an exchange that fails is passed over.
type PerpetualMarketStructureService struct {
	perpetualMarketStructureProxies []domaininterface.IPerpetualMarketStructureProxy
	sourceRequestTimeout            time.Duration
	// maximumConcurrentLookups keeps a batch of coins from bursting past the exchanges' free request rates.
	maximumConcurrentLookups int
}

func NewPerpetualMarketStructureService(
	perpetualMarketStructureProxies []domaininterface.IPerpetualMarketStructureProxy, sourceRequestTimeout time.Duration, maximumConcurrentLookups int,
) *PerpetualMarketStructureService {
	return &PerpetualMarketStructureService{
		perpetualMarketStructureProxies: perpetualMarketStructureProxies,
		sourceRequestTimeout:            sourceRequestTimeout,
		maximumConcurrentLookups:        max(1, maximumConcurrentLookups),
	}
}

// FindMarketStructure returns nil when no exchange answers for the coin.
func (perpetualMarketStructureService *PerpetualMarketStructureService) FindMarketStructure(
	executionContext context.Context, coinSymbol string,
) *vo.PerpetualMarketStructureVo {
	return perpetualMarketStructureService.findFirstListing(executionContext, coinSymbol)
}

// FindMarketStructures asks for several coins at once, a bounded number at a time; a coin no exchange answers for is absent.
func (perpetualMarketStructureService *PerpetualMarketStructureService) FindMarketStructures(
	executionContext context.Context, coinSymbols []string,
) map[string]vo.PerpetualMarketStructureVo {
	marketStructures := map[string]vo.PerpetualMarketStructureVo{}
	marketStructuresLock := sync.Mutex{}
	lookupSlots := make(chan struct{}, perpetualMarketStructureService.maximumConcurrentLookups)
	lookups := sync.WaitGroup{}
	for _, coinSymbol := range coinSymbols {
		lookups.Go(func() {
			lookupSlots <- struct{}{}
			defer func() { <-lookupSlots }()
			if marketStructure := perpetualMarketStructureService.findFirstListing(executionContext, coinSymbol); marketStructure != nil {
				marketStructuresLock.Lock()
				marketStructures[coinSymbol] = *marketStructure
				marketStructuresLock.Unlock()
			}
		})
	}
	lookups.Wait()

	return marketStructures
}

// findFirstListing asks each exchange in turn under its own deadline; an exchange that errs or lacks the coin is skipped.
func (perpetualMarketStructureService *PerpetualMarketStructureService) findFirstListing(
	executionContext context.Context, coinSymbol string,
) *vo.PerpetualMarketStructureVo {
	for _, perpetualMarketStructureProxy := range perpetualMarketStructureService.perpetualMarketStructureProxies {
		if executionContext.Err() != nil {
			return nil
		}
		sourceContext, cancelSource := context.WithTimeout(executionContext, perpetualMarketStructureService.sourceRequestTimeout)
		marketStructure, found, findError := perpetualMarketStructureProxy.FindMarketStructure(sourceContext, coinSymbol)
		cancelSource()
		if findError == nil && found {
			return &marketStructure
		}
	}

	return nil
}
