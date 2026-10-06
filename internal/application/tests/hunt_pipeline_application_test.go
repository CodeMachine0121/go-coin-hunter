package application_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/handler"
	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var roundStartedAt = time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC)

// huntPipelineWorld wires the four real services; the run store remembers every run so each step finds its upstream.
type huntPipelineWorld struct {
	huntPipelineApplication *application.HuntPipelineApplication
	runs                    []entities.PipelineRun
	runsLock                sync.Mutex
	informationSource       *mocks.MockIInformationSourceProxy
	coinFilterResults       *mocks.MockICoinFilterResultRepository
	analyst                 *mocks.MockICoinInsightAnalystProxy
	strategist              *mocks.MockIHuntVerdictStrategistProxy
	huntBoard               *mocks.MockIHuntBoardRepository
	storedFilterResults     []entities.CoinFilterResult
	storedInsights          []entities.CoinInsight
	insightSaveError        error
	boardRewrites           [][]entities.HuntBoardEntry
}

func (world *huntPipelineWorld) latestSucceeded(step string) (entities.PipelineRun, bool) {
	world.runsLock.Lock()
	defer world.runsLock.Unlock()
	for index := len(world.runs) - 1; index >= 0; index-- {
		if world.runs[index].Step == step && world.runs[index].Status == string(vo.PipelineRunStatusSucceeded) {
			return world.runs[index], true
		}
	}
	return entities.PipelineRun{}, false
}

func (world *huntPipelineWorld) stepsRun() []string {
	world.runsLock.Lock()
	defer world.runsLock.Unlock()
	steps := []string{}
	for _, run := range world.runs {
		steps = append(steps, run.Step+":"+run.TriggerSource)
	}
	return steps
}

func newHuntPipelineWorld(t *testing.T) *huntPipelineWorld {
	controller := gomock.NewController(t)
	world := &huntPipelineWorld{
		informationSource: mocks.NewMockIInformationSourceProxy(controller),
		coinFilterResults: mocks.NewMockICoinFilterResultRepository(controller),
		analyst:           mocks.NewMockICoinInsightAnalystProxy(controller),
		strategist:        mocks.NewMockIHuntVerdictStrategistProxy(controller),
		huntBoard:         mocks.NewMockIHuntBoardRepository(controller),
	}
	pipelineRuns := mocks.NewMockIPipelineRunRepository(controller)
	pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, run entities.PipelineRun) (entities.PipelineRun, error) {
		world.runsLock.Lock()
		defer world.runsLock.Unlock()
		run.ID = uint(len(world.runs) + 1)
		world.runs = append(world.runs, run)
		return run, nil
	}).AnyTimes()
	pipelineRuns.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, run entities.PipelineRun) error {
		world.runsLock.Lock()
		defer world.runsLock.Unlock()
		world.runs[run.ID-1] = run
		return nil
	}).AnyTimes()
	pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, step string) (entities.PipelineRun, bool, error) {
		run, found := world.latestSucceeded(step)
		return run, found, nil
	}).AnyTimes()

	outcomes := mocks.NewMockIInformationSourceOutcomeRepository(controller)
	outcomes.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	savedIntelligences := []entities.CoinIntelligence{}
	intelligences := mocks.NewMockICoinIntelligenceRepository(controller)
	intelligences.EXPECT().SaveNew(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, saved []entities.CoinIntelligence) error {
		savedIntelligences = append(savedIntelligences, saved...)
		return nil
	}).AnyTimes()
	intelligences.EXPECT().FindPublishedSince(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, time.Time) ([]entities.CoinIntelligence, error) {
		return savedIntelligences, nil
	}).AnyTimes()
	intelligences.EXPECT().FindDeclaredContractAddresses(gomock.Any(), gomock.Any()).Return(map[string]vo.TokenAddressVo{}, nil).AnyTimes()
	intelligences.EXPECT().FindByCoinSymbolsSince(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	storedCandidates := []entities.CoinCandidate{}
	candidates := mocks.NewMockICoinCandidateRepository(controller)
	candidates.EXPECT().CreateAll(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, created []entities.CoinCandidate) error {
		storedCandidates = append(storedCandidates, created...)
		return nil
	}).AnyTimes()
	candidates.EXPECT().FindByPipelineRunID(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, pipelineRunID uint) ([]entities.CoinCandidate, error) {
		return storedCandidates, nil
	}).AnyTimes()
	world.coinFilterResults.EXPECT().CreateAll(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, created []entities.CoinFilterResult) error {
		world.storedFilterResults = created
		return nil
	}).AnyTimes()
	world.coinFilterResults.EXPECT().FindByPipelineRunID(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, uint) ([]entities.CoinFilterResult, error) {
		return world.storedFilterResults, nil
	}).AnyTimes()
	insights := mocks.NewMockICoinInsightRepository(controller)
	insights.EXPECT().CreateAll(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, created []entities.CoinInsight) error {
		if world.insightSaveError != nil {
			return world.insightSaveError
		}
		world.storedInsights = created
		return nil
	}).AnyTimes()
	insights.EXPECT().FindByPipelineRunID(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, uint) ([]entities.CoinInsight, error) {
		return world.storedInsights, nil
	}).AnyTimes()
	verdicts := mocks.NewMockICoinVerdictRepository(controller)
	verdicts.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	marketData := mocks.NewMockICoinMarketDataProxy(controller)
	marketData.EXPECT().SourceName().Return("coinGeckoMarketData").AnyTimes()
	marketData.EXPECT().FindCoinMarketData(gomock.Any(), gomock.Any()).Return(map[string]vo.CoinMarketDataVo{"ZORA": healthyMarketData("zora")}, nil).AnyTimes()
	security := mocks.NewMockITokenSecurityProxy(controller)
	security.EXPECT().SupportsChain(gomock.Any()).Return(false).AnyTimes()
	listing := mocks.NewMockIPerpetualContractListingProxy(controller)
	listing.EXPECT().ExchangeName().Return("Bybit").AnyTimes()
	listing.EXPECT().FindUsdtPerpetualCoinSymbols(gomock.Any()).Return(map[string]bool{"ZORA": true}, nil).AnyTimes()
	unlocks := mocks.NewMockITokenUnlockScheduleProxy(controller)
	unlocks.EXPECT().FindTokenUnlockEvents(gomock.Any(), gomock.Any()).Return(map[string][]vo.TokenUnlockEventVo{}, nil).AnyTimes()
	news := mocks.NewMockICoinNewsProxy(controller)
	news.EXPECT().FindRecentHeadlines(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	market := mocks.NewMockIPerpetualMarketStructureProxy(controller)
	market.EXPECT().FindMarketStructure(gomock.Any(), gomock.Any()).Return(vo.PerpetualMarketStructureVo{ExchangeName: "Bybit", LastPrice: usd("0.05")}, true, nil).AnyTimes()
	world.informationSource.EXPECT().SourceName().Return("bybitAnnouncement").AnyTimes()

	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(roundStartedAt).AnyTimes()
	marketStructureService := service.NewPerpetualMarketStructureService([]domaininterface.IPerpetualMarketStructureProxy{market}, time.Second)
	world.huntPipelineApplication = application.NewHuntPipelineApplication(
		service.NewCoinDiscoveryService([]domaininterface.IInformationSourceProxy{world.informationSource}, pipelineRuns, outcomes, intelligences, candidates, clockProxy, discoveryPolicy()),
		service.NewCoinFilteringService(pipelineRuns, candidates, world.coinFilterResults,
			service.NewCoinProfileService(intelligences, []domaininterface.ICoinMarketDataProxy{marketData}, security,
				[]domaininterface.IPerpetualContractListingProxy{listing}, unlocks,
				service.NewPerpetualMarketStructureService(nil, time.Second), profileTiming(time.Second)),
			[]domaininterface.ICoinCandidateFilterHandler{handler.NewPerpetualContractListingFilterHandler()}, clockProxy),
		service.NewCoinInsightService(pipelineRuns, candidates, world.coinFilterResults, insights,
			service.NewCoinInsightMaterialService(intelligences, news, marketStructureService, insightPolicy()), world.analyst, clockProxy, insightPolicy()),
		service.NewHuntVerdictService(pipelineRuns, insights, verdicts, world.huntBoard, marketStructureService, world.strategist, clockProxy, huntVerdictPolicy()),
	)

	return world
}

// expectTheBoardRewritten answers every rewrite of the board with the given error and keeps what each rewrite held.
func (world *huntPipelineWorld) expectTheBoardRewritten(rewriteError error) {
	world.huntBoard.EXPECT().Rewrite(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, entries []entities.HuntBoardEntry) error {
		world.boardRewrites = append(world.boardRewrites, entries)
		return rewriteError
	}).MinTimes(1)
}

func (world *huntPipelineWorld) discoveryFinds(coinSymbols ...string) {
	informationItems := []vo.InformationItemVo{}
	for _, coinSymbol := range coinSymbols {
		informationItems = append(informationItems, vo.InformationItemVo{SourceName: "bybitAnnouncement", ExternalIdentifier: coinSymbol,
			DeclaredCoinSymbols: []string{coinSymbol}, PublishedAt: &roundStartedAt})
	}
	world.informationSource.EXPECT().FetchInformationItems(gomock.Any(), gomock.Any()).Return(informationItems, nil).AnyTimes()
}

func TestRunHuntRoundRunsEveryStepInOrder(t *testing.T) {
	world := newHuntPipelineWorld(t)
	world.discoveryFinds("ZORA")
	world.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).Return(bullishAnswer(), nil)
	world.strategist.EXPECT().SynthesizeVerdicts(gomock.Any(), gomock.Any()).Return([]vo.HuntVerdictAnswerVo{longAnswer("ZORA")}, nil)
	rewrittenBoard := []entities.HuntBoardEntry{}
	world.huntBoard.EXPECT().Rewrite(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, entries []entities.HuntBoardEntry) error {
		rewrittenBoard = entries
		return nil
	})

	huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceJob)
	require.NoError(t, roundError)

	assert.True(t, huntRound.Completed)
	assert.Empty(t, huntRound.StoppedStep)
	require.Len(t, huntRound.Steps, 4)
	assert.Equal(t, []string{"discovery:job", "filtering:job", "insight:job", "verdict:job"}, world.stepsRun())
	for index, step := range huntRound.Steps {
		assert.Equal(t, uint(index+1), step.ID)
		assert.Equal(t, string(vo.PipelineRunStatusSucceeded), step.Status)
	}
	assert.Equal(t, uint(1), *huntRound.Steps[1].TriggeredByPipelineRunID)
	assert.Equal(t, uint(3), *huntRound.Steps[3].TriggeredByPipelineRunID)
	require.Len(t, rewrittenBoard, 1)
	assert.Equal(t, "ZORA", rewrittenBoard[0].CoinSymbol)
}

func TestRunHuntRoundStopsAtTheFirstStepThatDoesNotSucceed(t *testing.T) {
	t.Run("discovery with no data empties the board", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		world.discoveryFinds()
		world.expectTheBoardRewritten(nil)

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceJob)
		require.NoError(t, roundError)

		assert.False(t, huntRound.Completed)
		assert.Equal(t, "discovery", huntRound.StoppedStep)
		assert.Equal(t, "探索未成功：noData", huntRound.StoppedReason)
		assert.Equal(t, []string{"discovery:job"}, world.stepsRun())
		assert.Len(t, huntRound.Steps, 1)
		assert.Equal(t, [][]entities.HuntBoardEntry{{}}, world.boardRewrites)
	})

	t.Run("filtering keeping nothing empties the board", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		world.discoveryFinds("DOUU")
		world.expectTheBoardRewritten(nil)

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceJob)
		require.NoError(t, roundError)

		assert.Equal(t, "filtering", huntRound.StoppedStep)
		assert.Equal(t, "過濾未成功：noData", huntRound.StoppedReason)
		assert.Equal(t, []string{"discovery:job", "filtering:job"}, world.stepsRun())
		assert.Equal(t, [][]entities.HuntBoardEntry{{}}, world.boardRewrites)
	})

	t.Run("a verdict without a bullish insight empties the board", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		world.discoveryFinds("ZORA")
		bearish := bullishAnswer()
		bearish.Direction = "bearish"
		world.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).Return(bearish, nil)
		world.expectTheBoardRewritten(nil)

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceJob)
		require.NoError(t, roundError)

		assert.Equal(t, "verdict", huntRound.StoppedStep)
		assert.Equal(t, "裁決未成功：noData", huntRound.StoppedReason)
		assert.NotEmpty(t, world.boardRewrites)
		for _, boardRewrite := range world.boardRewrites {
			assert.Empty(t, boardRewrite)
		}
	})

	t.Run("a board that cannot be emptied is told in the reason", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		world.discoveryFinds()
		world.expectTheBoardRewritten(errors.New("rewrite hunt board: disk full"))

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceJob)
		require.NoError(t, roundError)

		assert.Equal(t, "discovery", huntRound.StoppedStep)
		assert.Equal(t, "探索未成功：noData；清空獵捕結果表失敗：rewrite hunt board: disk full", huntRound.StoppedReason)
	})

	// A failed step never touches the board: the strict board mock refuses any rewrite here.
	t.Run("insight failing as a step error", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		world.discoveryFinds("ZORA")
		world.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).Return(bullishAnswer(), nil)
		world.insightSaveError = errors.New("disk full")

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceJob)
		require.NoError(t, roundError)

		assert.Equal(t, "insight", huntRound.StoppedStep)
		assert.Contains(t, huntRound.StoppedReason, "disk full")
		assert.Equal(t, []string{"discovery:job", "filtering:job", "insight:job"}, world.stepsRun())
		assert.Len(t, huntRound.Steps, 2)
	})

	t.Run("verdict failing", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		world.discoveryFinds("ZORA")
		world.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).Return(bullishAnswer(), nil)
		world.strategist.EXPECT().SynthesizeVerdicts(gomock.Any(), gomock.Any()).Return(nil, errors.New("overloaded"))

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceManual)
		require.NoError(t, roundError)

		assert.Equal(t, "verdict", huntRound.StoppedStep)
		assert.Equal(t, "裁決未成功：failed", huntRound.StoppedReason)
		assert.Equal(t, []string{"discovery:manual", "filtering:manual", "insight:manual", "verdict:manual"}, world.stepsRun())
		assert.Len(t, huntRound.Steps, 4)
	})
}

func TestRunHuntRoundStartsNoStepOnceAskedToStop(t *testing.T) {
	t.Run("asked to stop", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		stopped := make(chan struct{})
		close(stopped)

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), stopped, vo.PipelineRunTriggerSourceJob)
		require.NoError(t, roundError)

		assert.Equal(t, "discovery", huntRound.StoppedStep)
		assert.Equal(t, "探索未開始：服務關閉中", huntRound.StoppedReason)
		assert.Empty(t, world.stepsRun())
	})

	t.Run("an ended context", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		ended, cancel := context.WithCancel(context.Background())
		cancel()

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(ended, nil, vo.PipelineRunTriggerSourceJob)
		require.NoError(t, roundError)

		assert.Equal(t, "探索未開始：context canceled", huntRound.StoppedReason)
		assert.Empty(t, world.stepsRun())
	})

	t.Run("asked to stop during a step lets that step finish", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		stopRequested := make(chan struct{})
		world.informationSource.EXPECT().FetchInformationItems(gomock.Any(), gomock.Any()).DoAndReturn(
			func(sourceContext context.Context, _ int) ([]vo.InformationItemVo, error) {
				close(stopRequested)
				assert.NoError(t, sourceContext.Err())
				return []vo.InformationItemVo{{SourceName: "bybitAnnouncement", ExternalIdentifier: "ZORA", DeclaredCoinSymbols: []string{"ZORA"}, PublishedAt: &roundStartedAt}}, nil
			})

		huntRound, roundError := world.huntPipelineApplication.RunHuntRound(context.Background(), stopRequested, vo.PipelineRunTriggerSourceJob)
		require.NoError(t, roundError)

		assert.Equal(t, "filtering", huntRound.StoppedStep)
		assert.Equal(t, "過濾未開始：服務關閉中", huntRound.StoppedReason)
		require.Len(t, huntRound.Steps, 1)
		assert.Equal(t, string(vo.PipelineRunStatusSucceeded), huntRound.Steps[0].Status)
	})
}

func TestRunHuntRoundRefusesASecondRoundWhileOneIsRunning(t *testing.T) {
	world := newHuntPipelineWorld(t)
	// Both rounds discover nothing, so each empties the board.
	world.expectTheBoardRewritten(nil)
	inDiscovery := make(chan struct{})
	release := make(chan struct{})
	world.informationSource.EXPECT().FetchInformationItems(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, int) ([]vo.InformationItemVo, error) {
			close(inDiscovery)
			<-release
			return []vo.InformationItemVo{}, nil
		})
	firstRoundDone := make(chan struct{})
	go func() {
		defer close(firstRoundDone)
		_, _ = world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceJob)
	}()
	<-inDiscovery

	_, overlapError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceManual)

	assert.ErrorIs(t, overlapError, domains.ErrHuntRoundAlreadyRunning)
	assert.EqualError(t, overlapError, "已有獵捕回合進行中")
	close(release)
	<-firstRoundDone
	assert.Equal(t, []string{"discovery:job"}, world.stepsRun())

	world.informationSource.EXPECT().FetchInformationItems(gomock.Any(), gomock.Any()).Return([]vo.InformationItemVo{}, nil)
	_, laterError := world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceManual)
	assert.NoError(t, laterError, "a round may run again once the previous one ended")
}

// capturedLog collects what the standard logger writes during a test.
func capturedLog(t *testing.T) *bytes.Buffer {
	logBuffer := &bytes.Buffer{}
	previousWriter := log.Writer()
	log.SetOutput(logBuffer)
	t.Cleanup(func() { log.SetOutput(previousWriter) })
	return logBuffer
}

func TestRunHuntRoundLeavesOneLogLinePerRound(t *testing.T) {
	t.Run("a completed round", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		world.discoveryFinds("ZORA")
		world.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).Return(bullishAnswer(), nil)
		world.strategist.EXPECT().SynthesizeVerdicts(gomock.Any(), gomock.Any()).Return([]vo.HuntVerdictAnswerVo{longAnswer("ZORA")}, nil)
		world.huntBoard.EXPECT().Rewrite(gomock.Any(), gomock.Any()).Return(nil)
		logBuffer := capturedLog(t)

		_, _ = world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceManual)

		assert.Contains(t, logBuffer.String(), "hunt round (manual) completed: pipeline runs [1 2 3 4]")
		assert.Equal(t, 1, strings.Count(logBuffer.String(), "hunt round ("))
	})

	t.Run("a stopped round", func(t *testing.T) {
		world := newHuntPipelineWorld(t)
		world.discoveryFinds()
		world.expectTheBoardRewritten(nil)
		logBuffer := capturedLog(t)

		_, _ = world.huntPipelineApplication.RunHuntRound(context.Background(), nil, vo.PipelineRunTriggerSourceJob)

		assert.Contains(t, logBuffer.String(), "hunt round (job) stopped at discovery (探索未成功：noData): pipeline runs [1]")
	})
}
