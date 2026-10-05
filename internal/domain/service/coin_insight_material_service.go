package service

import (
	"context"
	"fmt"
	"time"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

const (
	newsUnavailableDataGap            = "查不到近期新聞"
	marketStructureUnavailableDataGap = "查不到永續合約市場結構"
)

// CoinInsightMaterialService gathers what the analyst is shown about each candidate. Missing material is a gap to
// report, never a reason to stop: the analyst can weigh what it lacks.
type CoinInsightMaterialService struct {
	coinIntelligenceRepository      domaininterface.ICoinIntelligenceRepository
	coinNewsProxy                   domaininterface.ICoinNewsProxy
	perpetualMarketStructureService *PerpetualMarketStructureService
	coinInsightPolicy               vo.CoinInsightPolicyVo
}

func NewCoinInsightMaterialService(
	coinIntelligenceRepository domaininterface.ICoinIntelligenceRepository,
	coinNewsProxy domaininterface.ICoinNewsProxy,
	perpetualMarketStructureService *PerpetualMarketStructureService,
	coinInsightPolicy vo.CoinInsightPolicyVo,
) *CoinInsightMaterialService {
	return &CoinInsightMaterialService{
		coinIntelligenceRepository:      coinIntelligenceRepository,
		coinNewsProxy:                   coinNewsProxy,
		perpetualMarketStructureService: perpetualMarketStructureService,
		coinInsightPolicy:               coinInsightPolicy,
	}
}

// GatherCoinInsightMaterials returns one material per filter result, in order; only a storage failure is an error.
func (coinInsightMaterialService *CoinInsightMaterialService) GatherCoinInsightMaterials(
	executionContext context.Context, coinFilterResults []entities.CoinFilterResult, gatheredAt time.Time,
) ([]vo.CoinInsightMaterialVo, error) {
	policy := coinInsightMaterialService.coinInsightPolicy
	coinSymbols := make([]string, 0, len(coinFilterResults))
	for _, coinFilterResult := range coinFilterResults {
		coinSymbols = append(coinSymbols, coinFilterResult.CoinSymbol)
	}
	coinIntelligences, intelligenceError := coinInsightMaterialService.coinIntelligenceRepository.FindByCoinSymbolsSince(
		executionContext, coinSymbols, gatheredAt.Add(-policy.IntelligenceWindow))
	if intelligenceError != nil {
		return nil, fmt.Errorf("find intelligence for insight material: %w", intelligenceError)
	}
	intelligenceHeadlinesBySymbol := map[string][]vo.HeadlineVo{}
	for _, coinIntelligence := range coinIntelligences {
		if len(intelligenceHeadlinesBySymbol[coinIntelligence.CoinSymbol]) < policy.MaximumIntelligenceHeadline {
			intelligenceHeadlinesBySymbol[coinIntelligence.CoinSymbol] = append(intelligenceHeadlinesBySymbol[coinIntelligence.CoinSymbol],
				vo.HeadlineVo{SourceName: coinIntelligence.SourceName, Title: coinIntelligence.Title, PublishedAt: coinIntelligence.PublishedAt})
		}
	}

	materials := make([]vo.CoinInsightMaterialVo, 0, len(coinFilterResults))
	for _, coinFilterResult := range coinFilterResults {
		material := vo.CoinInsightMaterialVo{
			CoinSymbol:            coinFilterResult.CoinSymbol,
			IntelligenceHeadlines: intelligenceHeadlinesBySymbol[coinFilterResult.CoinSymbol],
			FilterVerdicts:        []vo.FilterVerdictVo{},
			DataGaps:              []string{},
		}
		for _, verdict := range coinFilterResult.Verdicts {
			material.FilterVerdicts = append(material.FilterVerdicts, vo.FilterVerdictVo{
				FilterName: verdict.FilterName, Outcome: vo.FilterOutcomeVo(verdict.Outcome), Reason: verdict.Reason,
			})
		}

		newsContext, cancelNews := context.WithTimeout(executionContext, policy.SourceRequestTimeout)
		newsHeadlines, newsError := coinInsightMaterialService.coinNewsProxy.FindRecentHeadlines(
			newsContext, coinFilterResult.CoinSymbol, gatheredAt.Add(-policy.NewsLookback), policy.MaximumNewsHeadlines)
		cancelNews()
		material.NewsHeadlines = newsHeadlines
		if newsError != nil || len(newsHeadlines) == 0 {
			material.NewsHeadlines = nil
			material.DataGaps = append(material.DataGaps, newsUnavailableDataGap)
		}

		material.MarketStructure = coinInsightMaterialService.perpetualMarketStructureService.FindMarketStructure(executionContext, coinFilterResult.CoinSymbol)
		if material.MarketStructure == nil {
			material.DataGaps = append(material.DataGaps, marketStructureUnavailableDataGap)
		}

		materials = append(materials, material)
	}

	return materials, nil
}
