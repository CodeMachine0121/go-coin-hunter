package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/handler"
	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var filteringStartedAt = time.Date(2026, 10, 5, 14, 0, 0, 0, time.UTC)

const (
	latestDiscoveryRunID = uint(5)
	filteringRunID       = uint(9)
)

func usd(text string) *decimal.Decimal {
	value := decimal.RequireFromString(text)
	return &value
}

// healthyMarketData passes every market rule on its own.
func healthyMarketData(coinGeckoID string, contractAddresses ...vo.TokenAddressVo) vo.CoinMarketDataVo {
	return vo.CoinMarketDataVo{SourceName: "coinGeckoMarketData", CoinGeckoID: coinGeckoID, Name: coinGeckoID,
		FullyDilutedValuationUsd: usd("77000000"), DailyVolumeUsd: usd("9000000"),
		CirculatingSupply: usd("4500000000"), MaxSupply: usd("10000000000"), ContractAddresses: contractAddresses}
}

type coinFilteringUnderTest struct {
	coinFilteringApplication *application.CoinFilteringApplication
	pipelineRunRepository    *mocks.MockIPipelineRunRepository
	coinCandidateRepository  *mocks.MockICoinCandidateRepository
	coinFilterResults        *mocks.MockICoinFilterResultRepository
	coinIntelligences        *mocks.MockICoinIntelligenceRepository
	primaryMarketData        *mocks.MockICoinMarketDataProxy
	fallbackMarketData       *mocks.MockICoinMarketDataProxy
	tokenSecurity            *mocks.MockITokenSecurityProxy
	binanceListing           *mocks.MockIPerpetualContractListingProxy
	bybitListing             *mocks.MockIPerpetualContractListingProxy
	okxListing               *mocks.MockIPerpetualContractListingProxy
	tokenUnlocks             *mocks.MockITokenUnlockScheduleProxy
	createdPipelineRun       *entities.PipelineRun
	lastPipelineRunUpdate    *entities.PipelineRun
	savedCoinFilterResults   []entities.CoinFilterResult
}

func defaultFilterHandlers() []domaininterface.ICoinCandidateFilterHandler {
	return []domaininterface.ICoinCandidateFilterHandler{
		handler.NewSecurityCheckFilterHandler(decimal.RequireFromString("0.1")),
		handler.NewLiquidityThresholdFilterHandler(decimal.NewFromInt(1_000_000)),
		handler.NewFullyDilutedValuationFilterHandler(decimal.NewFromInt(10_000_000), decimal.NewFromInt(1_000_000_000)),
		handler.NewCirculatingRatioFilterHandler(decimal.RequireFromString("0.2")),
		handler.NewUnlockScheduleFilterHandler(14*24*time.Hour, decimal.RequireFromString("0.05")),
		handler.NewPerpetualContractListingFilterHandler(),
	}
}

// newCoinFilteringUnderTest wires the real services and handlers around a latest discovery holding the given candidates;
// every source answers empty until a test says otherwise.
func newCoinFilteringUnderTest(t *testing.T, candidateSymbols ...string) *coinFilteringUnderTest {
	controller := gomock.NewController(t)
	underTest := &coinFilteringUnderTest{
		pipelineRunRepository:   mocks.NewMockIPipelineRunRepository(controller),
		coinCandidateRepository: mocks.NewMockICoinCandidateRepository(controller),
		coinFilterResults:       mocks.NewMockICoinFilterResultRepository(controller),
		coinIntelligences:       mocks.NewMockICoinIntelligenceRepository(controller),
		primaryMarketData:       mocks.NewMockICoinMarketDataProxy(controller),
		fallbackMarketData:      mocks.NewMockICoinMarketDataProxy(controller),
		tokenSecurity:           mocks.NewMockITokenSecurityProxy(controller),
		binanceListing:          mocks.NewMockIPerpetualContractListingProxy(controller),
		bybitListing:            mocks.NewMockIPerpetualContractListingProxy(controller),
		okxListing:              mocks.NewMockIPerpetualContractListingProxy(controller),
		tokenUnlocks:            mocks.NewMockITokenUnlockScheduleProxy(controller),
	}
	coinCandidates := []entities.CoinCandidate{}
	for _, candidateSymbol := range candidateSymbols {
		coinCandidates = append(coinCandidates, entities.CoinCandidate{PipelineRunID: latestDiscoveryRunID, CoinSymbol: candidateSymbol})
	}

	underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepDiscovery)).
		Return(entities.PipelineRun{ID: latestDiscoveryRunID}, true, nil).AnyTimes()
	underTest.coinCandidateRepository.EXPECT().FindByPipelineRunID(gomock.Any(), latestDiscoveryRunID).Return(coinCandidates, nil).AnyTimes()
	underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, pipelineRun entities.PipelineRun) (entities.PipelineRun, error) {
			pipelineRun.ID = filteringRunID
			underTest.createdPipelineRun = &pipelineRun
			return pipelineRun, nil
		}).AnyTimes()
	underTest.pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, pipelineRun entities.PipelineRun) error {
			underTest.lastPipelineRunUpdate = &pipelineRun
			return nil
		}).AnyTimes()
	underTest.coinFilterResults.EXPECT().CreateAll(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, coinFilterResults []entities.CoinFilterResult) error {
			underTest.savedCoinFilterResults = coinFilterResults
			return nil
		}).AnyTimes()
	underTest.coinIntelligences.EXPECT().FindDeclaredContractAddresses(gomock.Any(), candidateSymbols).
		Return(map[string]vo.TokenAddressVo{}, nil).AnyTimes()
	underTest.primaryMarketData.EXPECT().SourceName().Return("coinGeckoMarketData").AnyTimes()
	underTest.fallbackMarketData.EXPECT().SourceName().Return("dexScreenerMarketData").AnyTimes()
	underTest.tokenSecurity.EXPECT().SourceName().Return("goPlusTokenSecurity").AnyTimes()
	underTest.tokenSecurity.EXPECT().SupportsChain(gomock.Any()).DoAndReturn(func(chainID string) bool {
		return chainID == vo.ChainEthereum || chainID == vo.ChainSolana
	}).AnyTimes()
	underTest.tokenUnlocks.EXPECT().SourceName().Return("defiLlamaUnlockSchedule").AnyTimes()
	for exchangeName, listing := range map[string]*mocks.MockIPerpetualContractListingProxy{
		"幣安": underTest.binanceListing, "Bybit": underTest.bybitListing, "OKX": underTest.okxListing,
	} {
		listing.EXPECT().ExchangeName().Return(exchangeName).AnyTimes()
	}

	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(filteringStartedAt).AnyTimes()
	underTest.coinFilteringApplication = application.NewCoinFilteringApplication(service.NewCoinFilteringService(
		underTest.pipelineRunRepository, underTest.coinCandidateRepository, underTest.coinFilterResults,
		service.NewCoinProfileService(underTest.coinIntelligences,
			[]domaininterface.ICoinMarketDataProxy{underTest.primaryMarketData, underTest.fallbackMarketData},
			underTest.tokenSecurity,
			[]domaininterface.IPerpetualContractListingProxy{underTest.binanceListing, underTest.bybitListing, underTest.okxListing},
			underTest.tokenUnlocks, time.Second),
		defaultFilterHandlers(), clockProxy))

	return underTest
}

func (underTest *coinFilteringUnderTest) answerMarketData(primary, fallback map[string]vo.CoinMarketDataVo) {
	underTest.primaryMarketData.EXPECT().FindCoinMarketData(gomock.Any(), gomock.Any()).Return(primary, nil).AnyTimes()
	underTest.fallbackMarketData.EXPECT().FindCoinMarketData(gomock.Any(), gomock.Any()).Return(fallback, nil).AnyTimes()
}

func (underTest *coinFilteringUnderTest) answerListings(binance, bybit, okx map[string]bool) {
	underTest.binanceListing.EXPECT().FindUsdtPerpetualCoinSymbols(gomock.Any()).Return(binance, nil).AnyTimes()
	underTest.bybitListing.EXPECT().FindUsdtPerpetualCoinSymbols(gomock.Any()).Return(bybit, nil).AnyTimes()
	underTest.okxListing.EXPECT().FindUsdtPerpetualCoinSymbols(gomock.Any()).Return(okx, nil).AnyTimes()
}

func (underTest *coinFilteringUnderTest) answerUnlocks(unlockEvents map[string][]vo.TokenUnlockEventVo) {
	underTest.tokenUnlocks.EXPECT().FindTokenUnlockEvents(gomock.Any(), gomock.Any()).Return(unlockEvents, nil).AnyTimes()
}

func (underTest *coinFilteringUnderTest) savedResult(t *testing.T, coinSymbol string) entities.CoinFilterResult {
	for _, coinFilterResult := range underTest.savedCoinFilterResults {
		if coinFilterResult.CoinSymbol == coinSymbol {
			return coinFilterResult
		}
	}
	t.Fatalf("no saved filter result for %s", coinSymbol)
	return entities.CoinFilterResult{}
}

func verdictOf(coinFilterResult entities.CoinFilterResult, filterName string) entities.CoinFilterVerdictRecord {
	for _, verdict := range coinFilterResult.Verdicts {
		if verdict.FilterName == filterName {
			return verdict
		}
	}
	return entities.CoinFilterVerdictRecord{}
}

func TestFilterCoinCandidatesKeepsOnlyCoinsNoRuleRejects(t *testing.T) {
	zoraContract := vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xzora"}
	pumpMarketData := healthyMarketData("pump-fun")
	pumpMarketData.FullyDilutedValuationUsd = usd("5200000000")
	underTest := newCoinFilteringUnderTest(t, "GRASS", "PUMP", "ZORA")
	underTest.answerMarketData(map[string]vo.CoinMarketDataVo{
		"ZORA": healthyMarketData("zora", zoraContract), "PUMP": pumpMarketData, "GRASS": healthyMarketData("grass"),
	}, map[string]vo.CoinMarketDataVo{})
	underTest.answerListings(map[string]bool{"PUMP": true, "GRASS": true}, map[string]bool{"ZORA": true}, map[string]bool{})
	underTest.answerUnlocks(map[string][]vo.TokenUnlockEventVo{"ZORA": {}, "PUMP": {}})
	underTest.tokenSecurity.EXPECT().FindTokenSecurity(gomock.Any(), zoraContract).Return(vo.TokenSecurityVo{}, true, nil)

	pipelineRun, filterError := underTest.coinFilteringApplication.FilterCoinCandidatesManually(context.Background())

	require.NoError(t, filterError)
	assert.Equal(t, string(vo.PipelineRunStatusSucceeded), pipelineRun.Status)
	assert.Equal(t, latestDiscoveryRunID, *pipelineRun.TriggeredByPipelineRunID)
	assert.Equal(t, string(vo.PipelineRunStepFiltering), pipelineRun.Step)
	assert.Equal(t, string(vo.PipelineRunTriggerSourceManual), pipelineRun.TriggerSource)

	zora := underTest.savedResult(t, "ZORA")
	assert.True(t, zora.IsKept)
	assert.Equal(t, "passed", verdictOf(zora, "securityCheck").Outcome)
	assert.Equal(t, entities.CoinFilterVerdictRecord{FilterName: "perpetualContractListing", Outcome: "passed", Reason: "已上 Bybit USDT 永續合約"},
		verdictOf(zora, "perpetualContractListing"))

	pump := underTest.savedResult(t, "PUMP")
	assert.False(t, pump.IsKept)
	assert.Equal(t, "完全稀釋估值 52 億美元高於上限 10 億美元", verdictOf(pump, "fullyDilutedValuation").Reason)

	grass := underTest.savedResult(t, "GRASS")
	assert.True(t, grass.IsKept)
	assert.Equal(t, entities.CoinFilterVerdictRecord{FilterName: "unlockSchedule", Outcome: "noData", Reason: "查不到解鎖時程"}, verdictOf(grass, "unlockSchedule"))
	assert.Len(t, grass.Verdicts, 6)
}

func TestFilterCoinCandidatesGathersProfilesFromTheRightSources(t *testing.T) {
	douuContract := vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "DouuMint"}
	underTest := newCoinFilteringUnderTest(t, "DOUU", "SUI", "AERO")
	underTest.coinIntelligences = nil
	controller := gomock.NewController(t)
	coinIntelligences := mocks.NewMockICoinIntelligenceRepository(controller)
	coinIntelligences.EXPECT().FindDeclaredContractAddresses(gomock.Any(), []string{"DOUU", "SUI", "AERO"}).
		Return(map[string]vo.TokenAddressVo{"DOUU": douuContract}, nil)
	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(filteringStartedAt).AnyTimes()
	application := application.NewCoinFilteringApplication(service.NewCoinFilteringService(
		underTest.pipelineRunRepository, underTest.coinCandidateRepository, underTest.coinFilterResults,
		service.NewCoinProfileService(coinIntelligences,
			[]domaininterface.ICoinMarketDataProxy{underTest.primaryMarketData, underTest.fallbackMarketData},
			underTest.tokenSecurity,
			[]domaininterface.IPerpetualContractListingProxy{underTest.binanceListing, underTest.bybitListing, underTest.okxListing},
			underTest.tokenUnlocks, time.Second),
		defaultFilterHandlers(), clockProxy))

	baseOnly := healthyMarketData("aerodrome-finance", vo.TokenAddressVo{ChainID: vo.ChainBase, Address: "0xaero"})
	underTest.primaryMarketData.EXPECT().FindCoinMarketData(gomock.Any(), []vo.CoinIdentityVo{
		{CoinSymbol: "DOUU", DeclaredContractAddress: &douuContract}, {CoinSymbol: "SUI"}, {CoinSymbol: "AERO"},
	}).Return(map[string]vo.CoinMarketDataVo{"SUI": healthyMarketData("sui"), "AERO": baseOnly}, nil)
	dexDouu := vo.CoinMarketDataVo{SourceName: "dexScreenerMarketData", FullyDilutedValuationUsd: usd("3000000")}
	underTest.fallbackMarketData.EXPECT().FindCoinMarketData(gomock.Any(), gomock.Any()).
		Return(map[string]vo.CoinMarketDataVo{"DOUU": dexDouu, "SUI": {SourceName: "dexScreenerMarketData"}}, nil)
	underTest.answerListings(map[string]bool{"DOUU": true, "SUI": true}, map[string]bool{"SUI": true}, map[string]bool{"AERO": true})
	underTest.tokenUnlocks.EXPECT().FindTokenUnlockEvents(gomock.Any(), []vo.CoinUnlockLookupVo{
		{CoinSymbol: "SUI", CoinGeckoID: "sui", Name: "sui"}, {CoinSymbol: "AERO", CoinGeckoID: "aerodrome-finance", Name: "aerodrome-finance"},
	}).Return(map[string][]vo.TokenUnlockEventVo{}, nil)
	// Only DOUU is checked: SUI has no contract, and AERO lives on a chain the security source cannot check.
	underTest.tokenSecurity.EXPECT().FindTokenSecurity(gomock.Any(), douuContract).Return(vo.TokenSecurityVo{IsMintable: true}, true, nil)

	_, filterError := application.FilterCoinCandidatesManually(context.Background())

	require.NoError(t, filterError)
	douu := underTest.savedResult(t, "DOUU")
	assert.Equal(t, "完全稀釋估值 300 萬美元低於下限 1,000 萬美元", verdictOf(douu, "fullyDilutedValuation").Reason)
	assert.Equal(t, "發行方仍可增發", verdictOf(douu, "securityCheck").Reason)
	assert.Equal(t, "已上 幣安、Bybit USDT 永續合約", verdictOf(underTest.savedResult(t, "SUI"), "perpetualContractListing").Reason)
	assert.Equal(t, "passed", verdictOf(underTest.savedResult(t, "SUI"), "liquidityThreshold").Outcome)
	assert.Equal(t, "沒有主流鏈上的合約位址", verdictOf(underTest.savedResult(t, "SUI"), "securityCheck").Reason)
	assert.Equal(t, "沒有主流鏈上的合約位址", verdictOf(underTest.savedResult(t, "AERO"), "securityCheck").Reason)
}

func TestFilterCoinCandidatesSkipsSecurityLookupsForUntradableCoins(t *testing.T) {
	underTest := newCoinFilteringUnderTest(t, "SCAM")
	underTest.answerMarketData(map[string]vo.CoinMarketDataVo{
		"SCAM": healthyMarketData("scam", vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xscam"}),
	}, map[string]vo.CoinMarketDataVo{})
	underTest.answerListings(map[string]bool{}, map[string]bool{}, map[string]bool{})
	underTest.answerUnlocks(map[string][]vo.TokenUnlockEventVo{})

	pipelineRun, filterError := underTest.coinFilteringApplication.FilterCoinCandidatesManually(context.Background())

	require.NoError(t, filterError)
	assert.Equal(t, string(vo.PipelineRunStatusNoData), pipelineRun.Status)
	scam := underTest.savedResult(t, "SCAM")
	assert.False(t, scam.IsKept)
	assert.Equal(t, entities.CoinFilterVerdictRecord{FilterName: "securityCheck", Outcome: "noData", Reason: "未上永續合約，未查詢安全資料"},
		verdictOf(scam, "securityCheck"))
	assert.Equal(t, "幣安、Bybit、OKX 皆無 USDT 永續合約", verdictOf(scam, "perpetualContractListing").Reason)
}

func TestFilterCoinCandidatesKeepsUnlockRejections(t *testing.T) {
	underTest := newCoinFilteringUnderTest(t, "ARB")
	marketData := healthyMarketData("arbitrum")
	marketData.CirculatingSupply = usd("1000000000")
	marketData.MaxSupply = usd("1000000000")
	underTest.answerMarketData(map[string]vo.CoinMarketDataVo{"ARB": marketData}, map[string]vo.CoinMarketDataVo{})
	underTest.answerListings(map[string]bool{"ARB": true}, map[string]bool{}, map[string]bool{})
	underTest.answerUnlocks(map[string][]vo.TokenUnlockEventVo{"ARB": {{UnlockAt: filteringStartedAt.Add(24 * time.Hour), Amount: decimal.NewFromInt(50_000_000)}}})

	_, filterError := underTest.coinFilteringApplication.FilterCoinCandidatesManually(context.Background())

	require.NoError(t, filterError)
	assert.Equal(t, "14 天內解鎖 5,000 萬，達流通量 5%（門檻 5%）", verdictOf(underTest.savedResult(t, "ARB"), "unlockSchedule").Reason)
}

func TestFilterCoinCandidatesRefusesWithoutASuccessfulDiscovery(t *testing.T) {
	controller := gomock.NewController(t)
	pipelineRunRepository := mocks.NewMockIPipelineRunRepository(controller)
	pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepDiscovery)).Return(entities.PipelineRun{}, false, nil)
	coinFilteringApplication := application.NewCoinFilteringApplication(service.NewCoinFilteringService(
		pipelineRunRepository, nil, nil, nil, nil, nil))

	_, filterError := coinFilteringApplication.FilterCoinCandidatesManually(context.Background())

	assert.ErrorIs(t, filterError, domains.ErrNoSucceededDiscoveryRun)
	assert.EqualError(t, filterError, "尚無成功的探索輪次")
}

func TestFilterCoinCandidatesFailsTheRoundWhenASourceIsDown(t *testing.T) {
	testCases := []struct {
		name       string
		breakIt    func(underTest *coinFilteringUnderTest)
		wantReason string
	}{
		{name: "a perpetual listing", breakIt: func(underTest *coinFilteringUnderTest) {
			underTest.answerMarketData(map[string]vo.CoinMarketDataVo{}, map[string]vo.CoinMarketDataVo{})
			underTest.binanceListing.EXPECT().FindUsdtPerpetualCoinSymbols(gomock.Any()).Return(map[string]bool{}, nil)
			underTest.bybitListing.EXPECT().FindUsdtPerpetualCoinSymbols(gomock.Any()).Return(nil, errors.New("連線逾時"))
			underTest.okxListing.EXPECT().FindUsdtPerpetualCoinSymbols(gomock.Any()).Return(map[string]bool{}, nil)
		}, wantReason: "資料來源無法取得：Bybit 永續合約清單（連線逾時）"},
		{name: "the market data", breakIt: func(underTest *coinFilteringUnderTest) {
			underTest.primaryMarketData.EXPECT().FindCoinMarketData(gomock.Any(), gomock.Any()).Return(nil, errors.New("429 too many requests"))
		}, wantReason: "資料來源無法取得：coinGeckoMarketData（429 too many requests）"},
		{name: "the security source", breakIt: func(underTest *coinFilteringUnderTest) {
			underTest.answerMarketData(map[string]vo.CoinMarketDataVo{"ZORA": healthyMarketData("zora", vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xzora"})}, nil)
			underTest.answerListings(map[string]bool{"ZORA": true}, map[string]bool{}, map[string]bool{})
			underTest.tokenSecurity.EXPECT().FindTokenSecurity(gomock.Any(), gomock.Any()).Return(vo.TokenSecurityVo{}, false, errors.New("連線逾時"))
		}, wantReason: "資料來源無法取得：goPlusTokenSecurity（連線逾時）"},
		{name: "the unlock schedule", breakIt: func(underTest *coinFilteringUnderTest) {
			underTest.answerMarketData(map[string]vo.CoinMarketDataVo{"ZORA": healthyMarketData("zora")}, nil)
			underTest.answerListings(map[string]bool{"ZORA": true}, map[string]bool{}, map[string]bool{})
			underTest.tokenUnlocks.EXPECT().FindTokenUnlockEvents(gomock.Any(), gomock.Any()).Return(nil, errors.New("連線逾時"))
		}, wantReason: "資料來源無法取得：defiLlamaUnlockSchedule（連線逾時）"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newCoinFilteringUnderTest(t, "ZORA")
			testCase.breakIt(underTest)

			pipelineRun, filterError := underTest.coinFilteringApplication.FilterCoinCandidatesManually(context.Background())

			require.NoError(t, filterError)
			assert.Equal(t, string(vo.PipelineRunStatusFailed), pipelineRun.Status)
			assert.Equal(t, testCase.wantReason, pipelineRun.FailureReason)
			assert.Equal(t, string(vo.PipelineRunStatusFailed), underTest.lastPipelineRunUpdate.Status)
			assert.Nil(t, underTest.savedCoinFilterResults)
		})
	}
}

func TestFilterCoinCandidatesSurfacesStorageFailures(t *testing.T) {
	t.Run("saving results fails the run and the call", func(t *testing.T) {
		controller := gomock.NewController(t)
		underTest := newCoinFilteringUnderTest(t, "ZORA")
		underTest.answerMarketData(map[string]vo.CoinMarketDataVo{}, map[string]vo.CoinMarketDataVo{})
		underTest.answerListings(map[string]bool{}, map[string]bool{}, map[string]bool{})
		failingResults := mocks.NewMockICoinFilterResultRepository(controller)
		failingResults.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(errors.New("disk full"))
		clockProxy := mocks.NewMockIClockProxy(controller)
		clockProxy.EXPECT().Now().Return(filteringStartedAt).AnyTimes()
		coinFilteringApplication := application.NewCoinFilteringApplication(service.NewCoinFilteringService(
			underTest.pipelineRunRepository, underTest.coinCandidateRepository, failingResults,
			service.NewCoinProfileService(underTest.coinIntelligences,
				[]domaininterface.ICoinMarketDataProxy{underTest.primaryMarketData, underTest.fallbackMarketData}, underTest.tokenSecurity,
				[]domaininterface.IPerpetualContractListingProxy{underTest.binanceListing, underTest.bybitListing, underTest.okxListing},
				underTest.tokenUnlocks, time.Second),
			defaultFilterHandlers(), clockProxy))

		_, filterError := coinFilteringApplication.FilterCoinCandidatesManually(context.Background())

		assert.ErrorContains(t, filterError, "disk full")
		assert.Equal(t, string(vo.PipelineRunStatusFailed), underTest.lastPipelineRunUpdate.Status)
	})

	testCases := []struct {
		name    string
		arrange func(pipelineRuns *mocks.MockIPipelineRunRepository, candidates *mocks.MockICoinCandidateRepository, intelligences *mocks.MockICoinIntelligenceRepository)
	}{
		{name: "finding the discovery", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, _ *mocks.MockICoinCandidateRepository, _ *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))
		}},
		{name: "finding the candidates", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, candidates *mocks.MockICoinCandidateRepository, _ *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 5}, true, nil)
			candidates.EXPECT().FindByPipelineRunID(gomock.Any(), uint(5)).Return(nil, errors.New("disk"))
		}},
		{name: "recording the run", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, candidates *mocks.MockICoinCandidateRepository, _ *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 5}, true, nil)
			candidates.EXPECT().FindByPipelineRunID(gomock.Any(), uint(5)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, errors.New("disk"))
		}},
		{name: "reading declared contracts, and recording that failure too", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, candidates *mocks.MockICoinCandidateRepository, intelligences *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 5}, true, nil)
			candidates.EXPECT().FindByPipelineRunID(gomock.Any(), uint(5)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 9}, nil)
			intelligences.EXPECT().FindDeclaredContractAddresses(gomock.Any(), gomock.Any()).Return(nil, errors.New("disk"))
			pipelineRuns.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("disk"))
		}},
		{name: "recording the conclusion", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, candidates *mocks.MockICoinCandidateRepository, intelligences *mocks.MockICoinIntelligenceRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 5}, true, nil)
			candidates.EXPECT().FindByPipelineRunID(gomock.Any(), uint(5)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 9}, nil)
			intelligences.EXPECT().FindDeclaredContractAddresses(gomock.Any(), gomock.Any()).Return(map[string]vo.TokenAddressVo{}, nil)
			pipelineRuns.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("disk"))
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			pipelineRuns := mocks.NewMockIPipelineRunRepository(controller)
			candidates := mocks.NewMockICoinCandidateRepository(controller)
			intelligences := mocks.NewMockICoinIntelligenceRepository(controller)
			results := mocks.NewMockICoinFilterResultRepository(controller)
			results.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			testCase.arrange(pipelineRuns, candidates, intelligences)
			clockProxy := mocks.NewMockIClockProxy(controller)
			clockProxy.EXPECT().Now().Return(filteringStartedAt).AnyTimes()
			coinFilteringApplication := application.NewCoinFilteringApplication(service.NewCoinFilteringService(
				pipelineRuns, candidates, results,
				service.NewCoinProfileService(intelligences, nil, nil, nil, nil, time.Second), nil, clockProxy))

			_, filterError := coinFilteringApplication.FilterCoinCandidatesManually(context.Background())

			assert.ErrorContains(t, filterError, "disk")
		})
	}
}

func TestGetLatestKeptCoinCandidates(t *testing.T) {
	t.Run("only the kept results of the newest successful filtering", func(t *testing.T) {
		underTest := newCoinFilteringUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepFiltering)).Return(entities.PipelineRun{ID: 2}, true, nil)
		underTest.coinFilterResults.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return([]entities.CoinFilterResult{
			{PipelineRunID: 2, CoinSymbol: "GRASS", IsKept: true, Verdicts: []entities.CoinFilterVerdictRecord{{FilterName: "unlockSchedule", Outcome: "noData", Reason: "查不到解鎖時程"}}},
			{PipelineRunID: 2, CoinSymbol: "PUMP", IsKept: false},
		}, nil)

		keptCandidates, findError := underTest.coinFilteringApplication.GetLatestKeptCoinCandidates(context.Background())

		require.NoError(t, findError)
		assert.Equal(t, []dto.CoinFilterResultDto{{PipelineRunID: 2, CoinSymbol: "GRASS", IsKept: true,
			Verdicts: []dto.CoinFilterVerdictDto{{FilterName: "unlockSchedule", Outcome: "noData", Reason: "查不到解鎖時程"}}}}, keptCandidates)
	})

	t.Run("never filtered successfully is empty", func(t *testing.T) {
		underTest := newCoinFilteringUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepFiltering)).Return(entities.PipelineRun{}, false, nil)

		keptCandidates, findError := underTest.coinFilteringApplication.GetLatestKeptCoinCandidates(context.Background())

		require.NoError(t, findError)
		assert.Equal(t, []dto.CoinFilterResultDto{}, keptCandidates)
	})

	t.Run("storage failing", func(t *testing.T) {
		for _, failResults := range []bool{false, true} {
			underTest := newCoinFilteringUnderTest(t)
			if failResults {
				underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepFiltering)).Return(entities.PipelineRun{ID: 2}, true, nil)
				underTest.coinFilterResults.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return(nil, errors.New("disk"))
			} else {
				underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepFiltering)).Return(entities.PipelineRun{}, false, errors.New("disk"))
			}

			_, findError := underTest.coinFilteringApplication.GetLatestKeptCoinCandidates(context.Background())

			assert.ErrorContains(t, findError, "disk")
		}
	})
}

func TestGetCoinFilterResultsOfPipelineRun(t *testing.T) {
	t.Run("every result, kept or not", func(t *testing.T) {
		underTest := newCoinFilteringUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(2)).Return(entities.PipelineRun{ID: 2}, nil)
		underTest.coinFilterResults.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return([]entities.CoinFilterResult{
			{PipelineRunID: 2, CoinSymbol: "GRASS", IsKept: true}, {PipelineRunID: 2, CoinSymbol: "PUMP", IsKept: false},
		}, nil)

		coinFilterResults, findError := underTest.coinFilteringApplication.GetCoinFilterResultsOfPipelineRun(context.Background(), 2)

		require.NoError(t, findError)
		require.Len(t, coinFilterResults, 2)
		assert.True(t, coinFilterResults[0].IsKept)
		assert.False(t, coinFilterResults[1].IsKept)
	})

	t.Run("an unknown run", func(t *testing.T) {
		underTest := newCoinFilteringUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(404)).Return(entities.PipelineRun{}, domains.ErrPipelineRunNotFound)

		_, findError := underTest.coinFilteringApplication.GetCoinFilterResultsOfPipelineRun(context.Background(), 404)

		assert.ErrorIs(t, findError, domains.ErrPipelineRunNotFound)
	})

	t.Run("storage failing", func(t *testing.T) {
		underTest := newCoinFilteringUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(2)).Return(entities.PipelineRun{ID: 2}, nil)
		underTest.coinFilterResults.EXPECT().FindByPipelineRunID(gomock.Any(), uint(2)).Return(nil, errors.New("disk"))

		_, findError := underTest.coinFilteringApplication.GetCoinFilterResultsOfPipelineRun(context.Background(), 2)

		assert.ErrorContains(t, findError, "disk")
	})
}
