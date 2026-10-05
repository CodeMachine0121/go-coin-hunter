package application_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var insightStartedAt = time.Date(2026, 10, 5, 16, 0, 0, 0, time.UTC)

const (
	latestFilteringRunID = uint(7)
	insightRunID         = uint(11)
)

func insightPolicy() vo.CoinInsightPolicyVo {
	return vo.CoinInsightPolicyVo{MaximumCoinsPerRound: 20, MaximumConcurrentAnalyses: 3, IntelligenceWindow: 72 * time.Hour,
		MaximumIntelligenceHeadline: 20, NewsLookback: 72 * time.Hour, MaximumNewsHeadlines: 10, SourceRequestTimeout: time.Second}
}

func bullishAnswer() vo.CoinInsightAnswerVo {
	return vo.CoinInsightAnswerVo{Direction: "bullish", Strength: 7, Catalyst: "幣安上新永續合約", Risks: []string{"解鎖"}, Evidence: []string{"持倉 +12%"}, DataGaps: []string{}}
}

type coinInsightUnderTest struct {
	coinInsightApplication *application.CoinInsightApplication
	pipelineRunRepository  *mocks.MockIPipelineRunRepository
	coinFilterResults      *mocks.MockICoinFilterResultRepository
	coinInsights           *mocks.MockICoinInsightRepository
	coinIntelligences      *mocks.MockICoinIntelligenceRepository
	coinNews               *mocks.MockICoinNewsProxy
	binanceMarket          *mocks.MockIPerpetualMarketStructureProxy
	bybitMarket            *mocks.MockIPerpetualMarketStructureProxy
	analyst                *mocks.MockICoinInsightAnalystProxy
	savedCoinInsights      []entities.CoinInsight
	lastPipelineRunUpdate  *entities.PipelineRun
	analyzedMaterials      sync.Map
}

// newCoinInsightUnderTest wires real services around a latest filtering that kept the given coins, each mentioned in
// that order; news and market structure answer for every coin unless a test says otherwise.
func newCoinInsightUnderTest(t *testing.T, policy vo.CoinInsightPolicyVo, keptSymbols ...string) *coinInsightUnderTest {
	controller := gomock.NewController(t)
	underTest := &coinInsightUnderTest{
		pipelineRunRepository: mocks.NewMockIPipelineRunRepository(controller),
		coinFilterResults:     mocks.NewMockICoinFilterResultRepository(controller),
		coinInsights:          mocks.NewMockICoinInsightRepository(controller),
		coinIntelligences:     mocks.NewMockICoinIntelligenceRepository(controller),
		coinNews:              mocks.NewMockICoinNewsProxy(controller),
		binanceMarket:         mocks.NewMockIPerpetualMarketStructureProxy(controller),
		bybitMarket:           mocks.NewMockIPerpetualMarketStructureProxy(controller),
		analyst:               mocks.NewMockICoinInsightAnalystProxy(controller),
	}
	coinCandidateRepository := mocks.NewMockICoinCandidateRepository(controller)
	discoveryRunID := uint(5)
	coinFilterResults := []entities.CoinFilterResult{{PipelineRunID: latestFilteringRunID, CoinSymbol: "DROPPED", IsKept: false}}
	coinCandidates := []entities.CoinCandidate{}
	for index, keptSymbol := range keptSymbols {
		coinFilterResults = append(coinFilterResults, entities.CoinFilterResult{PipelineRunID: latestFilteringRunID, CoinSymbol: keptSymbol, IsKept: true,
			Verdicts: []entities.CoinFilterVerdictRecord{{FilterName: "perpetualContractListing", Outcome: "passed", Reason: "已上 幣安 USDT 永續合約"}}})
		coinCandidates = append(coinCandidates, entities.CoinCandidate{CoinSymbol: keptSymbol, EarliestMentionedAt: insightStartedAt.Add(time.Duration(index) * time.Minute)})
	}

	underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepFiltering)).
		Return(entities.PipelineRun{ID: latestFilteringRunID, TriggeredByPipelineRunID: &discoveryRunID}, true, nil).AnyTimes()
	underTest.coinFilterResults.EXPECT().FindByPipelineRunID(gomock.Any(), latestFilteringRunID).Return(coinFilterResults, nil).AnyTimes()
	coinCandidateRepository.EXPECT().FindByPipelineRunID(gomock.Any(), discoveryRunID).Return(coinCandidates, nil).AnyTimes()
	underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, pipelineRun entities.PipelineRun) (entities.PipelineRun, error) {
			pipelineRun.ID = insightRunID
			return pipelineRun, nil
		}).AnyTimes()
	underTest.pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, pipelineRun entities.PipelineRun) error {
			underTest.lastPipelineRunUpdate = &pipelineRun
			return nil
		}).AnyTimes()
	underTest.coinInsights.EXPECT().CreateAll(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, coinInsights []entities.CoinInsight) error {
			underTest.savedCoinInsights = coinInsights
			return nil
		}).AnyTimes()
	underTest.coinIntelligences.EXPECT().FindByCoinSymbolsSince(gomock.Any(), gomock.Any(), insightStartedAt.Add(-72*time.Hour)).
		Return([]entities.CoinIntelligence{{SourceName: "binanceAnnouncement", CoinSymbol: "PENGU", Title: "Binance Futures Will Launch PENGUUSDT", PublishedAt: insightStartedAt}}, nil).AnyTimes()

	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(insightStartedAt).AnyTimes()
	underTest.coinInsightApplication = application.NewCoinInsightApplication(service.NewCoinInsightService(
		underTest.pipelineRunRepository, coinCandidateRepository, underTest.coinFilterResults, underTest.coinInsights,
		service.NewCoinInsightMaterialService(underTest.coinIntelligences, underTest.coinNews,
			[]domaininterface.IPerpetualMarketStructureProxy{underTest.binanceMarket, underTest.bybitMarket}, policy),
		underTest.analyst, clockProxy, policy))

	return underTest
}

func (underTest *coinInsightUnderTest) answerMaterialSources() {
	underTest.coinNews.EXPECT().FindRecentHeadlines(gomock.Any(), gomock.Any(), insightStartedAt.Add(-72*time.Hour), 10).
		Return([]vo.HeadlineVo{{SourceName: "CoinDesk", Title: "PENGU rallies", PublishedAt: insightStartedAt}}, nil).AnyTimes()
	underTest.binanceMarket.EXPECT().FindMarketStructure(gomock.Any(), gomock.Any()).
		Return(vo.PerpetualMarketStructureVo{ExchangeName: "幣安", FundingRate: usd("0.0001")}, true, nil).AnyTimes()
}

func (underTest *coinInsightUnderTest) savedInsight(t *testing.T, coinSymbol string) entities.CoinInsight {
	for _, coinInsight := range underTest.savedCoinInsights {
		if coinInsight.CoinSymbol == coinSymbol {
			return coinInsight
		}
	}
	t.Fatalf("no saved insight for %s", coinSymbol)
	return entities.CoinInsight{}
}

func TestAnalyzeCoinCandidatesProducesOneInsightPerKeptCoin(t *testing.T) {
	underTest := newCoinInsightUnderTest(t, insightPolicy(), "PENGU", "STRK")
	underTest.answerMaterialSources()
	underTest.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, material vo.CoinInsightMaterialVo) (vo.CoinInsightAnswerVo, error) {
			underTest.analyzedMaterials.Store(material.CoinSymbol, material)
			return bullishAnswer(), nil
		}).Times(2)

	pipelineRun, analyzeError := underTest.coinInsightApplication.AnalyzeCoinCandidatesManually(context.Background())

	require.NoError(t, analyzeError)
	assert.Equal(t, string(vo.PipelineRunStatusSucceeded), pipelineRun.Status)
	assert.Equal(t, string(vo.PipelineRunStepInsight), pipelineRun.Step)
	assert.Equal(t, string(vo.PipelineRunTriggerSourceManual), pipelineRun.TriggerSource)
	assert.Equal(t, latestFilteringRunID, *pipelineRun.TriggeredByPipelineRunID)
	assert.Equal(t, entities.CoinInsight{PipelineRunID: insightRunID, CoinSymbol: "PENGU", Succeeded: true, Direction: "bullish", Strength: 7,
		Catalyst: "幣安上新永續合約", Risks: []string{"解鎖"}, Evidence: []string{"持倉 +12%"}, DataGaps: []string{}, MarketStructureExchange: "幣安"},
		underTest.savedInsight(t, "PENGU"))
	assert.True(t, underTest.savedInsight(t, "STRK").Succeeded)

	penguMaterial, _ := underTest.analyzedMaterials.Load("PENGU")
	assert.Equal(t, vo.CoinInsightMaterialVo{
		CoinSymbol:            "PENGU",
		IntelligenceHeadlines: []vo.HeadlineVo{{SourceName: "binanceAnnouncement", Title: "Binance Futures Will Launch PENGUUSDT", PublishedAt: insightStartedAt}},
		NewsHeadlines:         []vo.HeadlineVo{{SourceName: "CoinDesk", Title: "PENGU rallies", PublishedAt: insightStartedAt}},
		MarketStructure:       &vo.PerpetualMarketStructureVo{ExchangeName: "幣安", FundingRate: usd("0.0001")},
		FilterVerdicts:        []vo.FilterVerdictVo{{FilterName: "perpetualContractListing", Outcome: vo.FilterOutcomePassed, Reason: "已上 幣安 USDT 永續合約"}},
		DataGaps:              []string{},
	}, penguMaterial)
}

func TestAnalyzeCoinCandidatesRefusesWithoutASuccessfulFiltering(t *testing.T) {
	controller := gomock.NewController(t)
	pipelineRunRepository := mocks.NewMockIPipelineRunRepository(controller)
	pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepFiltering)).Return(entities.PipelineRun{}, false, nil)
	coinInsightApplication := application.NewCoinInsightApplication(service.NewCoinInsightService(
		pipelineRunRepository, nil, nil, nil, nil, nil, nil, insightPolicy()))

	_, analyzeError := coinInsightApplication.AnalyzeCoinCandidatesManually(context.Background())

	assert.EqualError(t, analyzeError, "尚無成功的過濾輪次")
}

func TestAnalyzeCoinCandidatesAsksAgainOnceOnAnUnreadableAnswer(t *testing.T) {
	testCases := []struct {
		name           string
		penguAnswers   []error
		wantPenguCalls int
		wantSucceeded  bool
		wantReason     string
	}{
		{name: "the second answer is readable", penguAnswers: []error{fmt.Errorf("%w: unreadable answer", domains.ErrCoinInsightAnswerUnusable), nil},
			wantPenguCalls: 2, wantSucceeded: true},
		{name: "both answers are unreadable", penguAnswers: []error{domains.ErrCoinInsightAnswerUnusable, fmt.Errorf("%w: stopped with refusal", domains.ErrCoinInsightAnswerUnusable)},
			wantPenguCalls: 2, wantSucceeded: false, wantReason: "AI 回覆格式不合格"},
		{name: "a service error is not asked again", penguAnswers: []error{errors.New("ask claude for insight: 529 overloaded")},
			wantPenguCalls: 1, wantSucceeded: false, wantReason: "ask claude for insight: 529 overloaded"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newCoinInsightUnderTest(t, insightPolicy(), "PENGU", "STRK")
			underTest.answerMaterialSources()
			penguCalls := atomic.Int32{}
			underTest.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, material vo.CoinInsightMaterialVo) (vo.CoinInsightAnswerVo, error) {
					if material.CoinSymbol != "PENGU" {
						return bullishAnswer(), nil
					}
					answerError := testCase.penguAnswers[penguCalls.Add(1)-1]
					if answerError != nil {
						return vo.CoinInsightAnswerVo{}, answerError
					}
					return bullishAnswer(), nil
				}).AnyTimes()

			pipelineRun, analyzeError := underTest.coinInsightApplication.AnalyzeCoinCandidatesManually(context.Background())

			require.NoError(t, analyzeError)
			assert.Equal(t, int32(testCase.wantPenguCalls), penguCalls.Load())
			pengu := underTest.savedInsight(t, "PENGU")
			assert.Equal(t, testCase.wantSucceeded, pengu.Succeeded)
			assert.Equal(t, testCase.wantReason, pengu.FailureReason)
			assert.True(t, underTest.savedInsight(t, "STRK").Succeeded)
			assert.Equal(t, string(vo.PipelineRunStatusSucceeded), pipelineRun.Status)
		})
	}
}

func TestAnalyzeCoinCandidatesFailsTheRoundWhenEveryCoinFails(t *testing.T) {
	underTest := newCoinInsightUnderTest(t, insightPolicy(), "PENGU", "STRK")
	underTest.answerMaterialSources()
	underTest.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).Return(vo.CoinInsightAnswerVo{}, errors.New("401 invalid x-api-key")).Times(2)

	pipelineRun, analyzeError := underTest.coinInsightApplication.AnalyzeCoinCandidatesManually(context.Background())

	require.NoError(t, analyzeError)
	assert.Equal(t, string(vo.PipelineRunStatusFailed), pipelineRun.Status)
	assert.Equal(t, "所有候選幣分析失敗", pipelineRun.FailureReason)
	assert.Equal(t, entities.CoinInsight{PipelineRunID: insightRunID, CoinSymbol: "PENGU", FailureReason: "401 invalid x-api-key",
		Risks: []string{}, Evidence: []string{}, DataGaps: []string{}}, underTest.savedInsight(t, "PENGU"))
}

func TestAnalyzeCoinCandidatesReportsMissingMaterialAsDataGaps(t *testing.T) {
	underTest := newCoinInsightUnderTest(t, insightPolicy(), "STRK", "PONS")
	underTest.coinNews.EXPECT().FindRecentHeadlines(gomock.Any(), "STRK", gomock.Any(), gomock.Any()).Return([]vo.HeadlineVo{}, nil)
	underTest.coinNews.EXPECT().FindRecentHeadlines(gomock.Any(), "PONS", gomock.Any(), gomock.Any()).Return(nil, errors.New("503"))
	underTest.binanceMarket.EXPECT().FindMarketStructure(gomock.Any(), "STRK").Return(vo.PerpetualMarketStructureVo{}, false, errors.New("connection reset"))
	underTest.bybitMarket.EXPECT().FindMarketStructure(gomock.Any(), "STRK").Return(vo.PerpetualMarketStructureVo{ExchangeName: "Bybit"}, true, nil)
	underTest.binanceMarket.EXPECT().FindMarketStructure(gomock.Any(), "PONS").Return(vo.PerpetualMarketStructureVo{}, false, nil)
	underTest.bybitMarket.EXPECT().FindMarketStructure(gomock.Any(), "PONS").Return(vo.PerpetualMarketStructureVo{}, false, nil)
	underTest.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).Return(bullishAnswer(), nil).Times(2)

	_, analyzeError := underTest.coinInsightApplication.AnalyzeCoinCandidatesManually(context.Background())

	require.NoError(t, analyzeError)
	strk := underTest.savedInsight(t, "STRK")
	assert.True(t, strk.Succeeded)
	assert.Equal(t, []string{"查不到近期新聞"}, strk.DataGaps)
	assert.Equal(t, "Bybit", strk.MarketStructureExchange)
	pons := underTest.savedInsight(t, "PONS")
	assert.True(t, pons.Succeeded)
	assert.Equal(t, []string{"查不到近期新聞", "查不到永續合約市場結構"}, pons.DataGaps)
}

func TestAnalyzeCoinCandidatesStaysInsideItsCostLimits(t *testing.T) {
	keptSymbols := []string{}
	for index := range 25 {
		keptSymbols = append(keptSymbols, fmt.Sprintf("C%02d", index))
	}
	underTest := newCoinInsightUnderTest(t, insightPolicy(), keptSymbols...)
	underTest.answerMaterialSources()
	running, mostAtOnce := atomic.Int32{}, atomic.Int32{}
	underTest.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, vo.CoinInsightMaterialVo) (vo.CoinInsightAnswerVo, error) {
			current := running.Add(1)
			for seen := mostAtOnce.Load(); current > seen && !mostAtOnce.CompareAndSwap(seen, current); seen = mostAtOnce.Load() {
			}
			time.Sleep(10 * time.Millisecond)
			running.Add(-1)
			return bullishAnswer(), nil
		}).Times(20)

	_, analyzeError := underTest.coinInsightApplication.AnalyzeCoinCandidatesManually(context.Background())

	require.NoError(t, analyzeError)
	require.Len(t, underTest.savedCoinInsights, 20)
	assert.Equal(t, "C00", underTest.savedCoinInsights[0].CoinSymbol)
	assert.Equal(t, "C19", underTest.savedCoinInsights[19].CoinSymbol)
	assert.LessOrEqual(t, mostAtOnce.Load(), int32(3))
	assert.Greater(t, mostAtOnce.Load(), int32(1))
}

func TestAnalyzeCoinCandidatesSurfacesStorageFailures(t *testing.T) {
	t.Run("saving insights fails the run and the call", func(t *testing.T) {
		underTest := newCoinInsightUnderTest(t, insightPolicy(), "PENGU")
		underTest.answerMaterialSources()
		underTest.analyst.EXPECT().AnalyzeCoin(gomock.Any(), gomock.Any()).Return(bullishAnswer(), nil)
		controller := gomock.NewController(t)
		failingInsights := mocks.NewMockICoinInsightRepository(controller)
		failingInsights.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(errors.New("disk full"))
		candidates := mocks.NewMockICoinCandidateRepository(controller)
		candidates.EXPECT().FindByPipelineRunID(gomock.Any(), gomock.Any()).Return(nil, nil)
		clockProxy := mocks.NewMockIClockProxy(controller)
		clockProxy.EXPECT().Now().Return(insightStartedAt).AnyTimes()
		coinInsightApplication := application.NewCoinInsightApplication(service.NewCoinInsightService(
			underTest.pipelineRunRepository, candidates, underTest.coinFilterResults, failingInsights,
			service.NewCoinInsightMaterialService(underTest.coinIntelligences, underTest.coinNews,
				[]domaininterface.IPerpetualMarketStructureProxy{underTest.binanceMarket}, insightPolicy()),
			underTest.analyst, clockProxy, insightPolicy()))

		_, analyzeError := coinInsightApplication.AnalyzeCoinCandidatesManually(context.Background())

		assert.ErrorContains(t, analyzeError, "disk full")
		assert.Equal(t, string(vo.PipelineRunStatusFailed), underTest.lastPipelineRunUpdate.Status)
	})

	discoveryRunID := uint(5)
	testCases := []struct {
		name    string
		arrange func(pipelineRuns *mocks.MockIPipelineRunRepository, results *mocks.MockICoinFilterResultRepository,
			candidates *mocks.MockICoinCandidateRepository, intelligences *mocks.MockICoinIntelligenceRepository)
	}{
		{name: "finding the filtering", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, _ *mocks.MockICoinFilterResultRepository, _ *mocks.MockICoinCandidateRepository, _ *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))
		}},
		{name: "finding the results", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, results *mocks.MockICoinFilterResultRepository, _ *mocks.MockICoinCandidateRepository, _ *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 7}, true, nil)
			results.EXPECT().FindByPipelineRunID(gomock.Any(), uint(7)).Return(nil, errors.New("disk"))
		}},
		{name: "finding the candidates", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, results *mocks.MockICoinFilterResultRepository, candidates *mocks.MockICoinCandidateRepository, _ *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 7, TriggeredByPipelineRunID: &discoveryRunID}, true, nil)
			results.EXPECT().FindByPipelineRunID(gomock.Any(), uint(7)).Return(nil, nil)
			candidates.EXPECT().FindByPipelineRunID(gomock.Any(), discoveryRunID).Return(nil, errors.New("disk"))
		}},
		{name: "recording the run", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, results *mocks.MockICoinFilterResultRepository, _ *mocks.MockICoinCandidateRepository, _ *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 7}, true, nil)
			results.EXPECT().FindByPipelineRunID(gomock.Any(), uint(7)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, errors.New("disk"))
		}},
		{name: "reading intelligence, and recording that failure too", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, results *mocks.MockICoinFilterResultRepository, _ *mocks.MockICoinCandidateRepository, intelligences *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 7}, true, nil)
			results.EXPECT().FindByPipelineRunID(gomock.Any(), uint(7)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 11}, nil)
			intelligences.EXPECT().FindByCoinSymbolsSince(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("disk"))
			pipelineRuns.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("disk"))
		}},
		{name: "recording the conclusion", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, results *mocks.MockICoinFilterResultRepository, _ *mocks.MockICoinCandidateRepository, intelligences *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 7}, true, nil)
			results.EXPECT().FindByPipelineRunID(gomock.Any(), uint(7)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 11}, nil)
			intelligences.EXPECT().FindByCoinSymbolsSince(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			pipelineRuns.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("disk"))
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			pipelineRuns := mocks.NewMockIPipelineRunRepository(controller)
			results := mocks.NewMockICoinFilterResultRepository(controller)
			candidates := mocks.NewMockICoinCandidateRepository(controller)
			intelligences := mocks.NewMockICoinIntelligenceRepository(controller)
			insights := mocks.NewMockICoinInsightRepository(controller)
			insights.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			testCase.arrange(pipelineRuns, results, candidates, intelligences)
			clockProxy := mocks.NewMockIClockProxy(controller)
			clockProxy.EXPECT().Now().Return(insightStartedAt).AnyTimes()
			coinInsightApplication := application.NewCoinInsightApplication(service.NewCoinInsightService(
				pipelineRuns, candidates, results, insights,
				service.NewCoinInsightMaterialService(intelligences, nil, nil, insightPolicy()), nil, clockProxy, insightPolicy()))

			_, analyzeError := coinInsightApplication.AnalyzeCoinCandidatesManually(context.Background())

			assert.ErrorContains(t, analyzeError, "disk")
		})
	}
}

func TestGetCoinInsights(t *testing.T) {
	t.Run("the newest successful insight round", func(t *testing.T) {
		underTest := newCoinInsightUnderTest(t, insightPolicy())
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepInsight)).Return(entities.PipelineRun{ID: 2}, true, nil)
		underTest.coinInsights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return([]entities.CoinInsight{
			{PipelineRunID: 2, CoinSymbol: "PENGU", Succeeded: true, Direction: "bullish", Strength: 7, Risks: []string{"解鎖"}, Evidence: []string{}, DataGaps: []string{}},
		}, nil)

		coinInsights, findError := underTest.coinInsightApplication.GetLatestCoinInsights(context.Background())

		require.NoError(t, findError)
		assert.Equal(t, []dto.CoinInsightDto{{PipelineRunID: 2, CoinSymbol: "PENGU", Succeeded: true, Direction: "bullish", Strength: 7,
			Risks: []string{"解鎖"}, Evidence: []string{}, DataGaps: []string{}}}, coinInsights)
	})

	t.Run("never succeeded is empty", func(t *testing.T) {
		underTest := newCoinInsightUnderTest(t, insightPolicy())
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepInsight)).Return(entities.PipelineRun{}, false, nil)

		coinInsights, findError := underTest.coinInsightApplication.GetLatestCoinInsights(context.Background())

		require.NoError(t, findError)
		assert.Equal(t, []dto.CoinInsightDto{}, coinInsights)
	})

	t.Run("a run's results include failures", func(t *testing.T) {
		underTest := newCoinInsightUnderTest(t, insightPolicy())
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(2)).Return(entities.PipelineRun{ID: 2}, nil)
		underTest.coinInsights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return([]entities.CoinInsight{
			{PipelineRunID: 2, CoinSymbol: "PENGU", FailureReason: "AI 回覆格式不合格"},
			{PipelineRunID: 2, CoinSymbol: "STRK", Succeeded: true, Direction: "neutral", Strength: 4},
		}, nil)

		coinInsights, findError := underTest.coinInsightApplication.GetCoinInsightsOfPipelineRun(context.Background(), 2)

		require.NoError(t, findError)
		assert.Equal(t, []dto.CoinInsightDto{
			{PipelineRunID: 2, CoinSymbol: "PENGU", FailureReason: "AI 回覆格式不合格"},
			{PipelineRunID: 2, CoinSymbol: "STRK", Succeeded: true, Direction: "neutral", Strength: 4},
		}, coinInsights)
	})

	t.Run("an unknown run", func(t *testing.T) {
		underTest := newCoinInsightUnderTest(t, insightPolicy())
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(404)).Return(entities.PipelineRun{}, domains.ErrPipelineRunNotFound)

		_, findError := underTest.coinInsightApplication.GetCoinInsightsOfPipelineRun(context.Background(), 404)

		assert.ErrorIs(t, findError, domains.ErrPipelineRunNotFound)
	})

	t.Run("storage failing", func(t *testing.T) {
		for _, step := range []string{"latest run", "latest insights", "run insights"} {
			underTest := newCoinInsightUnderTest(t, insightPolicy())
			switch step {
			case "latest run":
				underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepInsight)).Return(entities.PipelineRun{}, false, errors.New("disk"))
				_, findError := underTest.coinInsightApplication.GetLatestCoinInsights(context.Background())
				assert.ErrorContains(t, findError, "disk")
			case "latest insights":
				underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepInsight)).Return(entities.PipelineRun{ID: 2}, true, nil)
				underTest.coinInsights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return(nil, errors.New("disk"))
				_, findError := underTest.coinInsightApplication.GetLatestCoinInsights(context.Background())
				assert.ErrorContains(t, findError, "disk")
			case "run insights":
				underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(2)).Return(entities.PipelineRun{ID: 2}, nil)
				underTest.coinInsights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return(nil, errors.New("disk"))
				_, findError := underTest.coinInsightApplication.GetCoinInsightsOfPipelineRun(context.Background(), 2)
				assert.ErrorContains(t, findError, "disk")
			}
		}
	})
}
