package application_test

import (
	"context"
	"errors"
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

var discoveryStartedAt = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

const discoveryPipelineRunID = uint(7)

func discoveryPolicy() vo.DiscoveryPolicyVo {
	return vo.DiscoveryPolicyVo{
		Window:               72 * time.Hour,
		ExcludedCoinSymbols:  []string{"BTC", "ETH", "BNB", "USDT", "USDC"},
		ItemLimitPerSource:   50,
		SourceRequestTimeout: time.Second,
	}
}

// fakeSource describes what one mocked information source answers.
type sourceAnswer struct {
	sourceName       string
	informationItems []vo.InformationItemVo
	failure          error
}

type coinDiscoveryUnderTest struct {
	coinDiscoveryApplication   *application.CoinDiscoveryApplication
	pipelineRunRepository      *mocks.MockIPipelineRunRepository
	coinIntelligenceRepository *mocks.MockICoinIntelligenceRepository
	coinCandidateRepository    *mocks.MockICoinCandidateRepository
	outcomeRepository          *mocks.MockIInformationSourceOutcomeRepository
	// savedCoinIntelligences is what SaveNew received; storedCoinIntelligences is what the window read returns on top of it.
	savedCoinIntelligences  []entities.CoinIntelligence
	storedCoinIntelligences []entities.CoinIntelligence
	createdCoinCandidates   []entities.CoinCandidate
	concludedPipelineRun    entities.PipelineRun
}

func newCoinDiscoveryUnderTest(t *testing.T, sourceAnswers []sourceAnswer) *coinDiscoveryUnderTest {
	controller := gomock.NewController(t)
	underTest := &coinDiscoveryUnderTest{
		pipelineRunRepository:      mocks.NewMockIPipelineRunRepository(controller),
		coinIntelligenceRepository: mocks.NewMockICoinIntelligenceRepository(controller),
		coinCandidateRepository:    mocks.NewMockICoinCandidateRepository(controller),
		outcomeRepository:          mocks.NewMockIInformationSourceOutcomeRepository(controller),
	}

	informationSourceProxies := []domaininterface.IInformationSourceProxy{}
	for _, answer := range sourceAnswers {
		informationSourceProxy := mocks.NewMockIInformationSourceProxy(controller)
		informationSourceProxy.EXPECT().SourceName().Return(answer.sourceName).AnyTimes()
		informationSourceProxy.EXPECT().FetchInformationItems(gomock.Any(), 50).
			Return(answer.informationItems, answer.failure).AnyTimes()
		informationSourceProxies = append(informationSourceProxies, informationSourceProxy)
	}

	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(discoveryStartedAt).AnyTimes()

	underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, pipelineRun entities.PipelineRun) (entities.PipelineRun, error) {
			pipelineRun.ID = discoveryPipelineRunID
			return pipelineRun, nil
		}).AnyTimes()
	underTest.pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, pipelineRun entities.PipelineRun) error {
			underTest.concludedPipelineRun = pipelineRun
			return nil
		}).AnyTimes()
	underTest.outcomeRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	underTest.coinIntelligenceRepository.EXPECT().SaveNew(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, coinIntelligences []entities.CoinIntelligence) error {
			underTest.savedCoinIntelligences = coinIntelligences
			return nil
		}).AnyTimes()
	underTest.coinIntelligenceRepository.EXPECT().FindPublishedSince(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, publishedSince time.Time) ([]entities.CoinIntelligence, error) {
			// Behaves like storage: a message about a coin from a source is kept once, the first copy winning.
			inWindow := []entities.CoinIntelligence{}
			keptMessages := map[string]bool{}
			for _, coinIntelligence := range append(underTest.storedCoinIntelligences, underTest.savedCoinIntelligences...) {
				messageKey := coinIntelligence.SourceName + "|" + coinIntelligence.ExternalIdentifier + "|" + coinIntelligence.CoinSymbol
				if keptMessages[messageKey] || coinIntelligence.PublishedAt.Before(publishedSince) {
					continue
				}
				keptMessages[messageKey] = true
				inWindow = append(inWindow, coinIntelligence)
			}
			return inWindow, nil
		}).AnyTimes()
	underTest.coinCandidateRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, coinCandidates []entities.CoinCandidate) error {
			underTest.createdCoinCandidates = coinCandidates
			return nil
		}).AnyTimes()

	underTest.coinDiscoveryApplication = application.NewCoinDiscoveryApplication(service.NewCoinDiscoveryService(
		informationSourceProxies, underTest.pipelineRunRepository, underTest.outcomeRepository,
		underTest.coinIntelligenceRepository, underTest.coinCandidateRepository, clockProxy, discoveryPolicy()))

	return underTest
}

func hoursBeforeStart(hours time.Duration) *time.Time {
	publishedAt := discoveryStartedAt.Add(-hours)
	return &publishedAt
}

func announcement(sourceName, identifier, title string, publishedAt *time.Time) vo.InformationItemVo {
	return vo.InformationItemVo{SourceName: sourceName, ExternalIdentifier: identifier, Title: title, Link: "https://example/" + identifier, PublishedAt: publishedAt}
}

func candidateSymbols(coinCandidates []entities.CoinCandidate) []string {
	coinSymbols := []string{}
	for _, coinCandidate := range coinCandidates {
		coinSymbols = append(coinSymbols, coinCandidate.CoinSymbol)
	}
	return coinSymbols
}

func TestDiscoverCoinsGroupsMentionsIntoCandidates(t *testing.T) {
	testCases := []struct {
		name                  string
		sourceAnswers         []sourceAnswer
		wantSymbols           []string
		wantFirstSourceCount  int
		wantFirstIntelligence int
	}{
		{
			name: "two sources naming the same coin make one candidate",
			sourceAnswers: []sourceAnswer{
				{sourceName: "binanceAnnouncement", informationItems: []vo.InformationItemVo{
					announcement("binanceAnnouncement", "a1", "Binance Will List Cotton (CT)", hoursBeforeStart(2))}},
				{sourceName: "bybitAnnouncement", informationItems: []vo.InformationItemVo{
					announcement("bybitAnnouncement", "b1", "New listing: CTUSDT Perpetual Contract", hoursBeforeStart(1))}},
			},
			wantSymbols: []string{"CT"}, wantFirstSourceCount: 2, wantFirstIntelligence: 2,
		},
		{
			name: "different coins each become a candidate",
			sourceAnswers: []sourceAnswer{
				{sourceName: "binanceAnnouncement", informationItems: []vo.InformationItemVo{
					announcement("binanceAnnouncement", "a1", "Binance Will List Cotton (CT)", hoursBeforeStart(2))}},
				{sourceName: "dexScreenerTokenProfile", informationItems: []vo.InformationItemVo{
					{SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "solana:pump", DeclaredCoinSymbols: []string{"pump"}, PublishedAt: hoursBeforeStart(3)}}},
			},
			wantSymbols: []string{"CT", "PUMP"}, wantFirstSourceCount: 1, wantFirstIntelligence: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newCoinDiscoveryUnderTest(t, testCase.sourceAnswers)

			pipelineRun, discoverError := underTest.coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

			require.NoError(t, discoverError)
			assert.Equal(t, string(vo.PipelineRunStatusSucceeded), pipelineRun.Status)
			assert.Equal(t, testCase.wantSymbols, candidateSymbols(underTest.createdCoinCandidates))
			assert.Equal(t, testCase.wantFirstSourceCount, underTest.createdCoinCandidates[0].SourceCount)
			assert.Equal(t, testCase.wantFirstIntelligence, underTest.createdCoinCandidates[0].IntelligenceCount)
		})
	}
}

func TestDiscoverCoinsKeepsAnAlreadyStoredMentionInWindow(t *testing.T) {
	alreadyStored := announcement("binanceAnnouncement", "a1", "Binance Will List Cotton (CT)", hoursBeforeStart(30))
	underTest := newCoinDiscoveryUnderTest(t, []sourceAnswer{
		{sourceName: "binanceAnnouncement", informationItems: []vo.InformationItemVo{alreadyStored}},
	})
	underTest.storedCoinIntelligences = []entities.CoinIntelligence{{
		SourceName: "binanceAnnouncement", ExternalIdentifier: "a1", CoinSymbol: "CT", PublishedAt: *hoursBeforeStart(30),
	}}

	_, discoverError := underTest.coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

	require.NoError(t, discoverError)
	assert.Equal(t, []string{"CT"}, candidateSymbols(underTest.createdCoinCandidates))
	assert.Equal(t, 1, underTest.createdCoinCandidates[0].IntelligenceCount)
}

func TestDiscoverCoinsWithEverySourceFailingKeepsNoCandidatesFromEarlierRounds(t *testing.T) {
	underTest := newCoinDiscoveryUnderTest(t, sixSources("every", errors.New("連線逾時")))
	underTest.storedCoinIntelligences = []entities.CoinIntelligence{{
		SourceName: "binanceAnnouncement", ExternalIdentifier: "a1", CoinSymbol: "CT", PublishedAt: *hoursBeforeStart(5),
	}}

	pipelineRun, discoverError := underTest.coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

	require.NoError(t, discoverError)
	assert.Equal(t, string(vo.PipelineRunStatusFailed), pipelineRun.Status)
	assert.Empty(t, underTest.createdCoinCandidates)
}

func TestDiscoverCoinsCutsOffASlowSourceWithoutHoldingUpTheOthers(t *testing.T) {
	controller := gomock.NewController(t)
	slowSource := mocks.NewMockIInformationSourceProxy(controller)
	slowSource.EXPECT().SourceName().Return("okxAnnouncement").AnyTimes()
	slowSource.EXPECT().FetchInformationItems(gomock.Any(), gomock.Any()).DoAndReturn(
		func(sourceContext context.Context, _ int) ([]vo.InformationItemVo, error) {
			<-sourceContext.Done()
			return nil, sourceContext.Err()
		})
	quickSource := mocks.NewMockIInformationSourceProxy(controller)
	quickSource.EXPECT().SourceName().Return("bybitAnnouncement").AnyTimes()
	quickSource.EXPECT().FetchInformationItems(gomock.Any(), gomock.Any()).Return([]vo.InformationItemVo{
		announcement("bybitAnnouncement", "b1", "New listing: CTUSDT Perpetual Contract", hoursBeforeStart(1)),
	}, nil)
	underTest := newCoinDiscoveryUnderTest(t, nil)
	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(discoveryStartedAt).AnyTimes()
	shortTimeoutPolicy := discoveryPolicy()
	shortTimeoutPolicy.SourceRequestTimeout = 50 * time.Millisecond
	coinDiscoveryApplication := application.NewCoinDiscoveryApplication(service.NewCoinDiscoveryService(
		[]domaininterface.IInformationSourceProxy{slowSource, quickSource}, underTest.pipelineRunRepository, underTest.outcomeRepository,
		underTest.coinIntelligenceRepository, underTest.coinCandidateRepository, clockProxy, shortTimeoutPolicy))

	pipelineRun, discoverError := coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

	require.NoError(t, discoverError)
	assert.Equal(t, string(vo.PipelineRunStatusSucceeded), pipelineRun.Status)
	assert.Equal(t, dto.InformationSourceOutcomeDto{SourceName: "okxAnnouncement", FailureReason: "連線逾時"}, pipelineRun.InformationSourceOutcomes[0])
	assert.True(t, pipelineRun.InformationSourceOutcomes[1].Succeeded)
	assert.Equal(t, []string{"CT"}, candidateSymbols(underTest.createdCoinCandidates))
}

func TestDiscoverCoinsHonorsTheDiscoveryWindow(t *testing.T) {
	testCases := []struct {
		name          string
		publishedAgo  time.Duration
		wantCandidate bool
	}{
		{name: "ten hours ago is inside", publishedAgo: 10 * time.Hour, wantCandidate: true},
		{name: "exactly seventy-two hours ago is inside", publishedAgo: 72 * time.Hour, wantCandidate: true},
		{name: "seventy-two hours and one minute ago is outside", publishedAgo: 72*time.Hour + time.Minute, wantCandidate: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newCoinDiscoveryUnderTest(t, []sourceAnswer{{sourceName: "binanceAnnouncement", informationItems: []vo.InformationItemVo{
				announcement("binanceAnnouncement", "a1", "Binance Will List Cotton (CT)", hoursBeforeStart(testCase.publishedAgo)),
			}}})

			_, discoverError := underTest.coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

			require.NoError(t, discoverError)
			assert.Equal(t, testCase.wantCandidate, len(underTest.createdCoinCandidates) == 1)
		})
	}
}

func TestDiscoverCoinsLeavesOutWhatIsNotANewCoin(t *testing.T) {
	testCases := []struct {
		name            string
		informationItem vo.InformationItemVo
		wantCoinSymbol  string
	}{
		{
			name:            "an excluded major coin",
			informationItem: vo.InformationItemVo{SourceName: "coinGeckoTrending", ExternalIdentifier: "ethereum", DeclaredCoinSymbols: []string{"ETH"}},
			wantCoinSymbol:  "ETH",
		},
		{
			name: "a stock perpetual contract",
			informationItem: vo.InformationItemVo{SourceName: "binancePerpetualContract", ExternalIdentifier: "NKEUSDT",
				DeclaredCoinSymbols: []string{"NKE"}, IsTraditionalAsset: true, PublishedAt: hoursBeforeStart(1)},
			wantCoinSymbol: "NKE",
		},
		{
			name:            "an announcement naming no coin",
			informationItem: announcement("binanceAnnouncement", "m1", "系統維護通知", hoursBeforeStart(1)),
			wantCoinSymbol:  "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newCoinDiscoveryUnderTest(t, []sourceAnswer{{sourceName: testCase.informationItem.SourceName,
				informationItems: []vo.InformationItemVo{testCase.informationItem}}})

			pipelineRun, discoverError := underTest.coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

			require.NoError(t, discoverError)
			assert.Empty(t, underTest.createdCoinCandidates)
			assert.Equal(t, string(vo.PipelineRunStatusNoData), pipelineRun.Status)
			require.Len(t, underTest.savedCoinIntelligences, 1)
			assert.Equal(t, testCase.wantCoinSymbol, underTest.savedCoinIntelligences[0].CoinSymbol)
		})
	}
}

func sixSources(failingSource string, failure error, coinSymbols ...string) []sourceAnswer {
	sourceNames := []string{"binanceAnnouncement", "binancePerpetualContract", "bybitAnnouncement",
		"okxAnnouncement", "coinGeckoTrending", "dexScreenerTokenProfile"}
	sourceAnswers := []sourceAnswer{}
	for index, sourceName := range sourceNames {
		answer := sourceAnswer{sourceName: sourceName, informationItems: []vo.InformationItemVo{}}
		if sourceName == failingSource || failingSource == "every" {
			answer.failure = failure
			answer.informationItems = nil
		} else if index < len(coinSymbols) {
			answer.informationItems = []vo.InformationItemVo{{SourceName: sourceName, ExternalIdentifier: coinSymbols[index],
				DeclaredCoinSymbols: []string{coinSymbols[index]}, PublishedAt: hoursBeforeStart(1)}}
		}
		sourceAnswers = append(sourceAnswers, answer)
	}
	return sourceAnswers
}

func TestDiscoverCoinsConcludesTheRunFromSourceOutcomes(t *testing.T) {
	timeout := errors.New("連線逾時")
	testCases := []struct {
		name              string
		sourceAnswers     []sourceAnswer
		wantStatus        vo.PipelineRunStatusVo
		wantFailureReason string
		wantCandidates    int
		wantFailedSources map[string]string
	}{
		{
			name: "every source succeeds with three coins", sourceAnswers: sixSources("", nil, "CT", "PUMP", "ZORA"),
			wantStatus: vo.PipelineRunStatusSucceeded, wantCandidates: 3, wantFailedSources: map[string]string{},
		},
		{
			name: "one source failing still succeeds", sourceAnswers: sixSources("okxAnnouncement", timeout, "CT", "PUMP"),
			wantStatus: vo.PipelineRunStatusSucceeded, wantCandidates: 2, wantFailedSources: map[string]string{"okxAnnouncement": "連線逾時"},
		},
		{
			name: "every source succeeds with nothing eligible", sourceAnswers: sixSources("", nil, "ETH"),
			wantStatus: vo.PipelineRunStatusNoData, wantCandidates: 0, wantFailedSources: map[string]string{},
		},
		{
			name: "every source failing fails the run", sourceAnswers: sixSources("every", timeout),
			wantStatus: vo.PipelineRunStatusFailed, wantFailureReason: domains.AllInformationSourcesFailedReason, wantCandidates: 0,
			wantFailedSources: map[string]string{"binanceAnnouncement": "連線逾時", "binancePerpetualContract": "連線逾時",
				"bybitAnnouncement": "連線逾時", "okxAnnouncement": "連線逾時", "coinGeckoTrending": "連線逾時", "dexScreenerTokenProfile": "連線逾時"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newCoinDiscoveryUnderTest(t, testCase.sourceAnswers)

			pipelineRun, discoverError := underTest.coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

			require.NoError(t, discoverError)
			assert.Equal(t, string(testCase.wantStatus), pipelineRun.Status)
			assert.Equal(t, testCase.wantFailureReason, pipelineRun.FailureReason)
			assert.Len(t, underTest.createdCoinCandidates, testCase.wantCandidates)
			assert.Equal(t, string(testCase.wantStatus), underTest.concludedPipelineRun.Status)
			require.Len(t, pipelineRun.InformationSourceOutcomes, 6)
			failedSources := map[string]string{}
			for _, outcome := range pipelineRun.InformationSourceOutcomes {
				if !outcome.Succeeded {
					failedSources[outcome.SourceName] = outcome.FailureReason
				}
			}
			assert.Equal(t, testCase.wantFailedSources, failedSources)
		})
	}
}

func TestDiscoverCoinsRecordsAManualTrigger(t *testing.T) {
	underTest := newCoinDiscoveryUnderTest(t, sixSources("", nil, "CT"))

	pipelineRun, discoverError := underTest.coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

	require.NoError(t, discoverError)
	assert.Equal(t, string(vo.PipelineRunTriggerSourceManual), pipelineRun.TriggerSource)
	assert.Equal(t, string(vo.PipelineRunStepDiscovery), pipelineRun.Step)
}

func TestDiscoverCoinsFailsWhenTheRunCannotBeRecorded(t *testing.T) {
	controller := gomock.NewController(t)
	pipelineRunRepository := mocks.NewMockIPipelineRunRepository(controller)
	pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, errors.New("disk full"))
	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(discoveryStartedAt).AnyTimes()
	coinDiscoveryApplication := application.NewCoinDiscoveryApplication(service.NewCoinDiscoveryService(
		nil, pipelineRunRepository, mocks.NewMockIInformationSourceOutcomeRepository(controller),
		mocks.NewMockICoinIntelligenceRepository(controller), mocks.NewMockICoinCandidateRepository(controller),
		clockProxy, discoveryPolicy()))

	_, discoverError := coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

	assert.ErrorContains(t, discoverError, "disk full")
}

// failingDiscoveryStorage wires discovery with every storage step succeeding except the one named.
func failingDiscoveryStorage(t *testing.T, failingStep string) (*application.CoinDiscoveryApplication, *entities.PipelineRun) {
	controller := gomock.NewController(t)
	stepError := func(step string) error {
		if step == failingStep {
			return errors.New("disk full")
		}
		return nil
	}
	lastUpdate := &entities.PipelineRun{}
	pipelineRunRepository := mocks.NewMockIPipelineRunRepository(controller)
	pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 1}, nil)
	pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, pipelineRun entities.PipelineRun) error {
		*lastUpdate = pipelineRun
		return stepError("update")
	}).Times(1)
	outcomeRepository := mocks.NewMockIInformationSourceOutcomeRepository(controller)
	outcomeRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(stepError("outcomes")).AnyTimes()
	coinIntelligenceRepository := mocks.NewMockICoinIntelligenceRepository(controller)
	coinIntelligenceRepository.EXPECT().SaveNew(gomock.Any(), gomock.Any()).Return(stepError("intelligences")).AnyTimes()
	coinIntelligenceRepository.EXPECT().FindPublishedSince(gomock.Any(), gomock.Any()).Return(nil, stepError("window")).AnyTimes()
	coinCandidateRepository := mocks.NewMockICoinCandidateRepository(controller)
	coinCandidateRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(stepError("candidates")).AnyTimes()
	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(discoveryStartedAt).AnyTimes()
	answeringSource := mocks.NewMockIInformationSourceProxy(controller)
	answeringSource.EXPECT().SourceName().Return("binanceAnnouncement").AnyTimes()
	answeringSource.EXPECT().FetchInformationItems(gomock.Any(), gomock.Any()).Return([]vo.InformationItemVo{}, nil)

	return application.NewCoinDiscoveryApplication(service.NewCoinDiscoveryService(
		[]domaininterface.IInformationSourceProxy{answeringSource}, pipelineRunRepository, outcomeRepository,
		coinIntelligenceRepository, coinCandidateRepository, clockProxy, discoveryPolicy())), lastUpdate
}

func TestDiscoverCoinsMarksTheRunFailedWhenSavingTheRoundFails(t *testing.T) {
	for _, failingStep := range []string{"outcomes", "intelligences", "window", "candidates"} {
		t.Run(failingStep, func(t *testing.T) {
			coinDiscoveryApplication, lastUpdate := failingDiscoveryStorage(t, failingStep)

			_, discoverError := coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

			assert.ErrorContains(t, discoverError, "disk full")
			assert.Equal(t, string(vo.PipelineRunStatusFailed), lastUpdate.Status)
			assert.Contains(t, lastUpdate.FailureReason, "disk full")
		})
	}
}

// The run is left running for the restart sweep: the failed conclusion is the only write attempted.
func TestDiscoverCoinsFailsWhenTheConclusionCannotBeRecorded(t *testing.T) {
	coinDiscoveryApplication, lastUpdate := failingDiscoveryStorage(t, "update")

	_, discoverError := coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

	assert.ErrorContains(t, discoverError, "disk full")
	assert.Equal(t, string(vo.PipelineRunStatusNoData), lastUpdate.Status)
	assert.Empty(t, lastUpdate.FailureReason)
}

func TestDiscoverCoinsReportsBothFailuresWhenTheFailureCannotBeRecorded(t *testing.T) {
	controller := gomock.NewController(t)
	pipelineRunRepository := mocks.NewMockIPipelineRunRepository(controller)
	pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 1}, nil)
	pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("database locked"))
	outcomeRepository := mocks.NewMockIInformationSourceOutcomeRepository(controller)
	outcomeRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(errors.New("disk full"))
	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(discoveryStartedAt).AnyTimes()
	coinDiscoveryApplication := application.NewCoinDiscoveryApplication(service.NewCoinDiscoveryService(
		nil, pipelineRunRepository, outcomeRepository, mocks.NewMockICoinIntelligenceRepository(controller),
		mocks.NewMockICoinCandidateRepository(controller), clockProxy, discoveryPolicy()))

	_, discoverError := coinDiscoveryApplication.DiscoverCoinsManually(context.Background())

	assert.ErrorContains(t, discoverError, "database locked")
	assert.ErrorContains(t, discoverError, "disk full")
}

func TestGetLatestCoinCandidatesFailsWhenStorageFails(t *testing.T) {
	underTest := newCoinDiscoveryUnderTest(t, nil)
	underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 2}, true, nil)
	underTest.coinCandidateRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return(nil, errors.New("disk"))

	_, findError := underTest.coinDiscoveryApplication.GetLatestCoinCandidates(context.Background())

	assert.ErrorContains(t, findError, "disk")
}

func TestGetLatestCoinCandidates(t *testing.T) {
	testCases := []struct {
		name        string
		latestRun   entities.PipelineRun
		found       bool
		candidates  []entities.CoinCandidate
		wantSymbols []string
	}{
		{
			name: "the newest successful run's candidates", latestRun: entities.PipelineRun{ID: 2}, found: true,
			candidates: []entities.CoinCandidate{{PipelineRunID: 2, CoinSymbol: "CT"}}, wantSymbols: []string{"CT"},
		},
		{name: "never succeeded is empty", found: false, wantSymbols: []string{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newCoinDiscoveryUnderTest(t, nil)
			underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepDiscovery)).
				Return(testCase.latestRun, testCase.found, nil)
			if testCase.found {
				underTest.coinCandidateRepository.EXPECT().FindByPipelineRunID(gomock.Any(), testCase.latestRun.ID).Return(testCase.candidates, nil)
			}

			coinCandidates, findError := underTest.coinDiscoveryApplication.GetLatestCoinCandidates(context.Background())

			require.NoError(t, findError)
			coinSymbols := []string{}
			for _, coinCandidate := range coinCandidates {
				coinSymbols = append(coinSymbols, coinCandidate.CoinSymbol)
			}
			assert.Equal(t, testCase.wantSymbols, coinSymbols)
		})
	}
}

func TestGetCoinIntelligencesOfPipelineRun(t *testing.T) {
	t.Run("a known run lists its intelligence", func(t *testing.T) {
		underTest := newCoinDiscoveryUnderTest(t, nil)
		publishedAt := *hoursBeforeStart(1)
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(1)).Return(entities.PipelineRun{ID: 1}, nil)
		underTest.coinIntelligenceRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(1)).Return([]entities.CoinIntelligence{
			{SourceName: "binanceAnnouncement", CoinSymbol: "CT", Title: "Binance Will List Cotton (CT)", Link: "https://example/a1", PublishedAt: publishedAt},
			{SourceName: "bybitAnnouncement", CoinSymbol: "CT", Title: "CTUSDT", Link: "https://example/b1", PublishedAt: publishedAt},
		}, nil)

		coinIntelligences, findError := underTest.coinDiscoveryApplication.GetCoinIntelligencesOfPipelineRun(context.Background(), 1)

		require.NoError(t, findError)
		assert.Equal(t, []dto.CoinIntelligenceDto{
			{SourceName: "binanceAnnouncement", CoinSymbol: "CT", Title: "Binance Will List Cotton (CT)", Link: "https://example/a1", PublishedAt: publishedAt},
			{SourceName: "bybitAnnouncement", CoinSymbol: "CT", Title: "CTUSDT", Link: "https://example/b1", PublishedAt: publishedAt},
		}, coinIntelligences)
	})

	t.Run("an unknown run is not found", func(t *testing.T) {
		underTest := newCoinDiscoveryUnderTest(t, nil)
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(404)).Return(entities.PipelineRun{}, domains.ErrPipelineRunNotFound)

		_, findError := underTest.coinDiscoveryApplication.GetCoinIntelligencesOfPipelineRun(context.Background(), 404)

		assert.ErrorIs(t, findError, domains.ErrPipelineRunNotFound)
	})
}
