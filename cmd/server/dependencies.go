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
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/marketdata"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/gin-gonic/gin"
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
		applicationConfig.Filtering.SourceRequestTimeout,
	)
}

// applications are built once by the composition root and shared by routes and background jobs.
type applications struct {
	coinDiscovery *application.CoinDiscoveryApplication
	coinFiltering *application.CoinFilteringApplication
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
