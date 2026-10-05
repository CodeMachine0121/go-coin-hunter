package service

import (
	"context"
	"time"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// PerpetualMarketStructureService answers "how does this coin's perpetual trade right now" from the exchanges in
// priority order: the first that lists the coin speaks for it, and an exchange that fails is passed over.
type PerpetualMarketStructureService struct {
	perpetualMarketStructureProxies []domaininterface.IPerpetualMarketStructureProxy
	sourceRequestTimeout            time.Duration
}

func NewPerpetualMarketStructureService(
	perpetualMarketStructureProxies []domaininterface.IPerpetualMarketStructureProxy, sourceRequestTimeout time.Duration,
) *PerpetualMarketStructureService {
	return &PerpetualMarketStructureService{
		perpetualMarketStructureProxies: perpetualMarketStructureProxies,
		sourceRequestTimeout:            sourceRequestTimeout,
	}
}

// FindMarketStructure returns nil when no exchange answers for the coin.
func (perpetualMarketStructureService *PerpetualMarketStructureService) FindMarketStructure(
	executionContext context.Context, coinSymbol string,
) *vo.PerpetualMarketStructureVo {
	for _, perpetualMarketStructureProxy := range perpetualMarketStructureService.perpetualMarketStructureProxies {
		sourceContext, cancelSource := context.WithTimeout(executionContext, perpetualMarketStructureService.sourceRequestTimeout)
		marketStructure, found, findError := perpetualMarketStructureProxy.FindMarketStructure(sourceContext, coinSymbol)
		cancelSource()
		if findError == nil && found {
			return &marketStructure
		}
	}

	return nil
}
