package main

import (
	"net/http"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/config"
	"github.com/CodeMachine0121/go-coin-hunter/internal/controller"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/handler"
	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/clock"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/informationsource"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/insight"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/marketdata"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/news"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/verdict"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// informationSourcesFor is where a new free information source is plugged in: one more line in this list.
func informationSourcesFor(discoveryConfig config.DiscoveryConfig) []domaininterface.IInformationSourceProxy {
	// No client-wide timeout: each source call runs under the discovery's own per-source deadline.
	httpClient := &http.Client{}

	return []domaininterface.IInformationSourceProxy{
		informationsource.NewBinanceAnnouncementInformationSourceProxy(httpClient, discoveryConfig.BinanceWebBaseUrl),
		informationsource.NewBinancePerpetualContractInformationSourceProxy(httpClient, discoveryConfig.BinanceFuturesUrl),
		informationsource.NewBybitAnnouncementInformationSourceProxy(httpClient, discoveryConfig.BybitBaseUrl),
		informationsource.NewOkxAnnouncementInformationSourceProxy(httpClient, discoveryConfig.OkxBaseUrl),
		informationsource.NewCoinGeckoTrendingInformationSourceProxy(httpClient, discoveryConfig.CoinGeckoBaseUrl),
		informationsource.NewDexScreenerTokenProfileInformationSourceProxy(httpClient, discoveryConfig.DexScreenerBaseUrl),
	}
}

// filterHandlersFor is where a filtering rule is plugged in: one more handler in this list.
func filterHandlersFor(filteringConfig config.FilteringConfig) []domaininterface.ICoinCandidateFilterHandler {
	return []domaininterface.ICoinCandidateFilterHandler{
		handler.NewSecurityCheckFilterHandler(filteringConfig.MaximumTaxRate),
		handler.NewLiquidityThresholdFilterHandler(filteringConfig.MinimumDailyVolumeUsd),
		handler.NewFullyDilutedValuationFilterHandler(filteringConfig.MinimumFullyDilutedValuationUsd, filteringConfig.MaximumFullyDilutedValuationUsd),
		handler.NewCirculatingRatioFilterHandler(filteringConfig.MinimumCirculatingRatio),
		handler.NewUnlockScheduleFilterHandler(filteringConfig.UnlockLookahead, filteringConfig.MaximumUnlockRatioOfCirculating),
		handler.NewPerpetualContractListingFilterHandler(),
	}
}

// coinProfileServiceFor wires the filtering data sources; market data sources are listed in priority order.
func coinProfileServiceFor(database *gorm.DB, applicationConfig config.ApplicationConfig) *service.CoinProfileService {
	// No client-wide timeout: each call runs under the service's own per-source deadline.
	httpClient := &http.Client{}

	return service.NewCoinProfileService(
		persistence.NewCoinIntelligenceRepository(database),
		[]domaininterface.ICoinMarketDataProxy{
			marketdata.NewCoinGeckoCoinMarketDataProxy(httpClient, applicationConfig.Discovery.CoinGeckoBaseUrl),
			marketdata.NewDexScreenerCoinMarketDataProxy(httpClient, applicationConfig.Discovery.DexScreenerBaseUrl),
		},
		marketdata.NewGoPlusTokenSecurityProxy(httpClient, applicationConfig.Filtering.GoPlusBaseUrl, applicationConfig.Filtering.TokenSecurityRequestInterval),
		[]domaininterface.IPerpetualContractListingProxy{
			marketdata.NewBinancePerpetualContractListingProxy(httpClient, applicationConfig.Discovery.BinanceFuturesUrl),
			marketdata.NewBybitPerpetualContractListingProxy(httpClient, applicationConfig.Discovery.BybitBaseUrl),
			marketdata.NewOkxPerpetualContractListingProxy(httpClient, applicationConfig.Discovery.OkxBaseUrl),
		},
		marketdata.NewDefiLlamaTokenUnlockScheduleProxy(httpClient, applicationConfig.Filtering.DefiLlamaDatasetsBaseUrl),
		vo.CoinProfileTimingVo{
			SourceRequestTimeout:    applicationConfig.Filtering.SourceRequestTimeout,
			RoundBaseBudget:         applicationConfig.Filtering.RoundBaseBudget,
			SecurityLookupAllowance: applicationConfig.Filtering.TokenSecurityRequestInterval,
		},
	)
}

// perpetualMarketStructureProxiesFor lists the exchanges in the order the insight rules prefer: Binance, Bybit, OKX.
func perpetualMarketStructureProxiesFor(httpClient *http.Client, discoveryConfig config.DiscoveryConfig) []domaininterface.IPerpetualMarketStructureProxy {
	return []domaininterface.IPerpetualMarketStructureProxy{
		marketdata.NewBinancePerpetualMarketStructureProxy(httpClient, discoveryConfig.BinanceFuturesUrl),
		marketdata.NewBybitPerpetualMarketStructureProxy(httpClient, discoveryConfig.BybitBaseUrl),
		marketdata.NewOkxPerpetualMarketStructureProxy(httpClient, discoveryConfig.OkxBaseUrl),
	}
}

// coinInsightPolicyFor shows the analyst the intelligence of the discovery window itself, so the two never disagree.
func coinInsightPolicyFor(applicationConfig config.ApplicationConfig) vo.CoinInsightPolicyVo {
	return vo.CoinInsightPolicyVo{
		MaximumCoinsPerRound:        applicationConfig.Insight.MaximumCoinsPerRound,
		MaximumConcurrentAnalyses:   applicationConfig.Insight.MaximumConcurrentAnalyses,
		IntelligenceWindow:          applicationConfig.Discovery.Window,
		MaximumIntelligenceHeadline: applicationConfig.Insight.MaximumIntelligenceHeadlines,
		NewsLookback:                applicationConfig.Insight.NewsLookback,
		MaximumNewsHeadlines:        applicationConfig.Insight.MaximumNewsHeadlines,
		SourceRequestTimeout:        applicationConfig.Insight.MaterialSourceTimeout,
	}
}

// coinInsightServiceFor wires the analyst and its material sources; market structure exchanges are listed in priority order.
func coinInsightServiceFor(database *gorm.DB, applicationConfig config.ApplicationConfig, clockProxy *clock.SystemClockProxy) *service.CoinInsightService {
	httpClient := &http.Client{}
	coinInsightPolicy := coinInsightPolicyFor(applicationConfig)

	return service.NewCoinInsightService(
		persistence.NewPipelineRunRepository(database),
		persistence.NewCoinCandidateRepository(database),
		persistence.NewCoinFilterResultRepository(database),
		persistence.NewCoinInsightRepository(database),
		service.NewCoinInsightMaterialService(
			persistence.NewCoinIntelligenceRepository(database),
			news.NewGoogleNewsCoinNewsProxy(httpClient, applicationConfig.Insight.GoogleNewsBaseUrl),
			perpetualMarketStructureProxiesFor(httpClient, applicationConfig.Discovery),
			coinInsightPolicy,
		),
		insight.NewClaudeCoinInsightAnalystProxy(
			applicationConfig.Insight.AnthropicApiKey, applicationConfig.Insight.AnthropicBaseUrl,
			applicationConfig.Insight.Model, applicationConfig.Insight.Effort, applicationConfig.Insight.AnalysisTimeout),
		clockProxy,
		coinInsightPolicy,
	)
}

// huntVerdictPolicyFor holds the safe ranges every verdict is clamped into: leverage 1-5, position up to 10%,
// stop loss 1-50% and take profit 1-200% of the latest price.
func huntVerdictPolicyFor(verdictConfig config.VerdictConfig) vo.HuntVerdictPolicyVo {
	return vo.HuntVerdictPolicyVo{
		MinimumLeverage:            1,
		MaximumLeverage:            5,
		MaximumPositionSizePercent: decimal.NewFromInt(10),
		MinimumStopLossPercent:     decimal.NewFromInt(1),
		MaximumStopLossPercent:     decimal.NewFromInt(50),
		MinimumTakeProfitPercent:   decimal.NewFromInt(1),
		MaximumTakeProfitPercent:   decimal.NewFromInt(200),
		MarketSourceTimeout:        verdictConfig.MarketSourceTimeout,
	}
}

// applications are built once by the composition root and shared by routes and background jobs.
type applications struct {
	coinDiscovery *application.CoinDiscoveryApplication
	coinFiltering *application.CoinFilteringApplication
	coinInsight   *application.CoinInsightApplication
	huntVerdict   *application.HuntVerdictApplication
	pipelineRun   *application.PipelineRunApplication
}

func applicationsFor(database *gorm.DB, applicationConfig config.ApplicationConfig) applications {
	clockProxy := clock.NewSystemClockProxy()
	pipelineRunRepository := persistence.NewPipelineRunRepository(database)
	coinCandidateRepository := persistence.NewCoinCandidateRepository(database)

	return applications{
		coinDiscovery: application.NewCoinDiscoveryApplication(service.NewCoinDiscoveryService(
			informationSourcesFor(applicationConfig.Discovery),
			pipelineRunRepository,
			persistence.NewInformationSourceOutcomeRepository(database),
			persistence.NewCoinIntelligenceRepository(database),
			coinCandidateRepository,
			clockProxy,
			vo.DiscoveryPolicyVo{
				Window:               applicationConfig.Discovery.Window,
				ExcludedCoinSymbols:  applicationConfig.Discovery.ExcludedCoinSymbols,
				ItemLimitPerSource:   applicationConfig.Discovery.ItemLimitPerSource,
				SourceRequestTimeout: applicationConfig.Discovery.SourceRequestTimeout,
			},
		)),
		coinFiltering: application.NewCoinFilteringApplication(service.NewCoinFilteringService(
			pipelineRunRepository,
			coinCandidateRepository,
			persistence.NewCoinFilterResultRepository(database),
			coinProfileServiceFor(database, applicationConfig),
			filterHandlersFor(applicationConfig.Filtering),
			clockProxy,
		)),
		coinInsight: application.NewCoinInsightApplication(coinInsightServiceFor(database, applicationConfig, clockProxy)),
		huntVerdict: application.NewHuntVerdictApplication(service.NewHuntVerdictService(
			pipelineRunRepository,
			persistence.NewCoinInsightRepository(database),
			persistence.NewCoinVerdictRepository(database),
			persistence.NewHuntBoardRepository(database),
			perpetualMarketStructureProxiesFor(&http.Client{}, applicationConfig.Discovery),
			verdict.NewClaudeHuntVerdictStrategistProxy(applicationConfig.Insight.AnthropicApiKey, applicationConfig.Insight.AnthropicBaseUrl,
				applicationConfig.Verdict.Model, applicationConfig.Verdict.Effort, applicationConfig.Verdict.SynthesisTimeout),
			clockProxy,
			huntVerdictPolicyFor(applicationConfig.Verdict),
		)),
		pipelineRun: application.NewPipelineRunApplication(service.NewPipelineRunService(pipelineRunRepository, clockProxy)),
	}
}

func registerRoutes(engine *gin.Engine, builtApplications applications) {
	engine.GET("/health", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{"status": "Healthy"})
	})

	coinDiscoveryController := controller.NewCoinDiscoveryController(builtApplications.coinDiscovery)
	engine.POST("/coin-discoveries", coinDiscoveryController.DiscoverCoins)
	engine.GET("/coin-candidates/latest", coinDiscoveryController.GetLatestCoinCandidates)
	engine.GET("/pipeline-runs/:pipelineRunId/coin-intelligences", coinDiscoveryController.GetCoinIntelligencesOfPipelineRun)

	coinFilteringController := controller.NewCoinFilteringController(builtApplications.coinFiltering)
	engine.POST("/coin-filterings", coinFilteringController.FilterCoinCandidates)
	engine.GET("/coin-filter-results/latest-kept", coinFilteringController.GetLatestKeptCoinCandidates)
	engine.GET("/pipeline-runs/:pipelineRunId/coin-filter-results", coinFilteringController.GetCoinFilterResultsOfPipelineRun)

	coinInsightController := controller.NewCoinInsightController(builtApplications.coinInsight)
	engine.POST("/coin-insights", coinInsightController.AnalyzeCoinCandidates)
	engine.GET("/coin-insights/latest", coinInsightController.GetLatestCoinInsights)
	engine.GET("/pipeline-runs/:pipelineRunId/coin-insights", coinInsightController.GetCoinInsightsOfPipelineRun)

	huntVerdictController := controller.NewHuntVerdictController(builtApplications.huntVerdict)
	engine.POST("/hunt-verdicts", huntVerdictController.SynthesizeHuntVerdicts)
	engine.GET("/hunt-board", huntVerdictController.GetHuntBoard)
	engine.GET("/pipeline-runs/:pipelineRunId/coin-verdicts", huntVerdictController.GetCoinVerdictsOfPipelineRun)

	pipelineRunController := controller.NewPipelineRunController(builtApplications.pipelineRun)
	engine.GET("/pipeline-runs", pipelineRunController.GetPipelineRuns)
}

// backgroundJobsFor returns no jobs when the global switch is off, which is how StartAll is disabled.
func backgroundJobsFor(applicationConfig config.ApplicationConfig, _ applications) []domaininterface.IBackgroundJob {
	if !applicationConfig.BackgroundJobsEnabled {
		return nil
	}

	return []domaininterface.IBackgroundJob{}
}
