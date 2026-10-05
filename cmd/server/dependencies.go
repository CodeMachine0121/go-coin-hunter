package main

import (
	"net/http"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/config"
	"github.com/CodeMachine0121/go-coin-hunter/internal/controller"
	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/clock"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/informationsource"
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

// applications are built once by the composition root and shared by routes and background jobs.
type applications struct {
	coinDiscovery *application.CoinDiscoveryApplication
	pipelineRun   *application.PipelineRunApplication
}

func applicationsFor(database *gorm.DB, applicationConfig config.ApplicationConfig) applications {
	clockProxy := clock.NewSystemClockProxy()
	pipelineRunRepository := persistence.NewPipelineRunRepository(database)

	return applications{
		coinDiscovery: application.NewCoinDiscoveryApplication(service.NewCoinDiscoveryService(
			informationSourcesFor(applicationConfig.Discovery),
			pipelineRunRepository,
			persistence.NewInformationSourceOutcomeRepository(database),
			persistence.NewCoinIntelligenceRepository(database),
			persistence.NewCoinCandidateRepository(database),
			clockProxy,
			vo.DiscoveryPolicyVo{
				Window:               applicationConfig.Discovery.Window,
				ExcludedCoinSymbols:  applicationConfig.Discovery.ExcludedCoinSymbols,
				ItemLimitPerSource:   applicationConfig.Discovery.ItemLimitPerSource,
				SourceRequestTimeout: applicationConfig.Discovery.SourceRequestTimeout,
			},
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
