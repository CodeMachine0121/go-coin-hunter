package application_test

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var verdictStartedAt = time.Date(2026, 10, 5, 18, 0, 0, 0, time.UTC)

const (
	latestInsightRunID = uint(9)
	verdictRunID       = uint(13)
)

func huntVerdictPolicy() vo.HuntVerdictPolicyVo {
	return vo.HuntVerdictPolicyVo{MinimumLeverage: 1, MaximumLeverage: 5, MaximumPositionSizePercent: decimal.NewFromInt(10),
		MinimumStopLossPercent: decimal.NewFromInt(1), MaximumStopLossPercent: decimal.NewFromInt(50),
		MinimumTakeProfitPercent: decimal.NewFromInt(1), MaximumTakeProfitPercent: decimal.NewFromInt(200)}
}

func longAnswer(coinSymbol string) vo.HuntVerdictAnswerVo {
	return vo.HuntVerdictAnswerVo{CoinSymbol: coinSymbol, Action: "long", Confidence: 70, Leverage: 3, PositionSizePercent: decimal.NewFromInt(5),
		StopLossPercent: decimal.NewFromInt(10), TakeProfitPercent: decimal.NewFromInt(30), Rationale: "幣安上新合約", ConflictResolution: "無明顯矛盾"}
}

type huntVerdictUnderTest struct {
	huntVerdictApplication *application.HuntVerdictApplication
	pipelineRunRepository  *mocks.MockIPipelineRunRepository
	coinInsights           *mocks.MockICoinInsightRepository
	coinVerdicts           *mocks.MockICoinVerdictRepository
	huntBoard              *mocks.MockIHuntBoardRepository
	binanceMarket          *mocks.MockIPerpetualMarketStructureProxy
	bybitMarket            *mocks.MockIPerpetualMarketStructureProxy
	strategist             *mocks.MockIHuntVerdictStrategistProxy
	rewrittenBoard         []entities.HuntBoardEntry
	savedCoinVerdicts      []entities.CoinVerdict
	shownMaterials         []vo.HuntVerdictMaterialVo
	lastPipelineRunUpdate  *entities.PipelineRun
}

// newHuntVerdictUnderTest wires the real service around a latest insight round with these successful insights (plus one
// failed insight that must never be shown); every coin trades on Binance at 0.01 unless a test says otherwise.
func newHuntVerdictUnderTest(t *testing.T, analyzedSymbols ...string) *huntVerdictUnderTest {
	controller := gomock.NewController(t)
	underTest := &huntVerdictUnderTest{
		pipelineRunRepository: mocks.NewMockIPipelineRunRepository(controller),
		coinInsights:          mocks.NewMockICoinInsightRepository(controller),
		coinVerdicts:          mocks.NewMockICoinVerdictRepository(controller),
		huntBoard:             mocks.NewMockIHuntBoardRepository(controller),
		binanceMarket:         mocks.NewMockIPerpetualMarketStructureProxy(controller),
		bybitMarket:           mocks.NewMockIPerpetualMarketStructureProxy(controller),
		strategist:            mocks.NewMockIHuntVerdictStrategistProxy(controller),
	}
	coinInsights := []entities.CoinInsight{{PipelineRunID: latestInsightRunID, CoinSymbol: "FAILED", FailureReason: "AI 回覆格式不合格"}}
	for _, analyzedSymbol := range analyzedSymbols {
		coinInsights = append(coinInsights, entities.CoinInsight{PipelineRunID: latestInsightRunID, CoinSymbol: analyzedSymbol, Succeeded: true,
			Direction: "bullish", Strength: 7, Catalyst: "幣安上新合約", Risks: []string{"解鎖"}, Evidence: []string{"持倉 +12%"}, DataGaps: []string{}})
	}

	underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepInsight)).
		Return(entities.PipelineRun{ID: latestInsightRunID}, true, nil).AnyTimes()
	underTest.coinInsights.EXPECT().FindByPipelineRunID(gomock.Any(), latestInsightRunID).Return(coinInsights, nil).AnyTimes()
	underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, pipelineRun entities.PipelineRun) (entities.PipelineRun, error) {
			pipelineRun.ID = verdictRunID
			return pipelineRun, nil
		}).AnyTimes()
	underTest.pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, pipelineRun entities.PipelineRun) error {
			underTest.lastPipelineRunUpdate = &pipelineRun
			return nil
		}).AnyTimes()
	underTest.coinVerdicts.EXPECT().CreateAll(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, coinVerdicts []entities.CoinVerdict) error {
			underTest.savedCoinVerdicts = coinVerdicts
			return nil
		}).AnyTimes()
	underTest.huntBoard.EXPECT().Rewrite(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, huntBoardEntries []entities.HuntBoardEntry) error {
			underTest.rewrittenBoard = huntBoardEntries
			return nil
		}).AnyTimes()

	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(verdictStartedAt).AnyTimes()
	underTest.huntVerdictApplication = application.NewHuntVerdictApplication(service.NewHuntVerdictService(
		underTest.pipelineRunRepository, underTest.coinInsights, underTest.coinVerdicts, underTest.huntBoard,
		service.NewPerpetualMarketStructureService([]domaininterface.IPerpetualMarketStructureProxy{underTest.binanceMarket, underTest.bybitMarket}, time.Second),
		underTest.strategist, clockProxy, huntVerdictPolicy()))

	return underTest
}

func (underTest *huntVerdictUnderTest) everyCoinOnBinance() {
	underTest.binanceMarket.EXPECT().FindMarketStructure(gomock.Any(), gomock.Any()).
		Return(vo.PerpetualMarketStructureVo{ExchangeName: "幣安", LastPrice: usd("0.01")}, true, nil).AnyTimes()
}

func (underTest *huntVerdictUnderTest) strategistAnswers(answers []vo.HuntVerdictAnswerVo, answerErrors ...error) {
	call := 0
	underTest.strategist.EXPECT().SynthesizeVerdicts(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, materials []vo.HuntVerdictMaterialVo) ([]vo.HuntVerdictAnswerVo, error) {
			underTest.shownMaterials = materials
			call++
			if call <= len(answerErrors) && answerErrors[call-1] != nil {
				return nil, answerErrors[call-1]
			}
			return answers, nil
		}).Times(max(1, len(answerErrors)))
}

func boardSymbols(huntBoardEntries []entities.HuntBoardEntry) []string {
	coinSymbols := []string{}
	for _, huntBoardEntry := range huntBoardEntries {
		coinSymbols = append(coinSymbols, huntBoardEntry.CoinSymbol)
	}
	return coinSymbols
}

func TestSynthesizeHuntVerdictsRewritesTheBoardWithThisRound(t *testing.T) {
	underTest := newHuntVerdictUnderTest(t, "BTC", "ETH")
	underTest.everyCoinOnBinance()
	underTest.strategistAnswers([]vo.HuntVerdictAnswerVo{longAnswer("BTC"), longAnswer("ETH")})

	pipelineRun, synthesizeError := underTest.huntVerdictApplication.SynthesizeHuntVerdictsManually(context.Background())

	require.NoError(t, synthesizeError)
	assert.Equal(t, string(vo.PipelineRunStatusSucceeded), pipelineRun.Status)
	assert.Equal(t, string(vo.PipelineRunStepVerdict), pipelineRun.Step)
	assert.Equal(t, string(vo.PipelineRunTriggerSourceManual), pipelineRun.TriggerSource)
	assert.Equal(t, latestInsightRunID, *pipelineRun.TriggeredByPipelineRunID)
	assert.Equal(t, []string{"BTC", "ETH"}, boardSymbols(underTest.rewrittenBoard))
	for _, huntBoardEntry := range underTest.rewrittenBoard {
		assert.Equal(t, verdictStartedAt, huntBoardEntry.CalculatedAt)
		assert.Equal(t, verdictRunID, huntBoardEntry.PipelineRunID)
		assert.Equal(t, "long", huntBoardEntry.Action)
		assert.Equal(t, "0.009", text(huntBoardEntry.StopLossPrice))
	}
	assert.Len(t, underTest.savedCoinVerdicts, 2)
	require.Len(t, underTest.shownMaterials, 2)
	assert.Equal(t, vo.HuntVerdictMaterialVo{CoinSymbol: "BTC", Direction: "bullish", Strength: 7, Catalyst: "幣安上新合約", Risks: []string{"解鎖"},
		Evidence: []string{"持倉 +12%"}, DataGaps: []string{}, MarketStructure: &vo.PerpetualMarketStructureVo{ExchangeName: "幣安", LastPrice: usd("0.01")}},
		underTest.shownMaterials[0])
}

func text(value *decimal.Decimal) string {
	if value == nil {
		return "none"
	}
	return value.String()
}

func TestSynthesizeHuntVerdictsLeavesTheBoardWhenTheStrategistCannotAnswer(t *testing.T) {
	testCases := []struct {
		name         string
		answerErrors []error
		wantReason   string
	}{
		{name: "two unreadable answers", answerErrors: []error{fmt.Errorf("%w: unreadable answer", domains.ErrHuntVerdictAnswerUnusable), domains.ErrHuntVerdictAnswerUnusable},
			wantReason: "AI 回覆格式不合格"},
		{name: "a failing service is not asked again", answerErrors: []error{errors.New("ask claude for verdicts: 529 overloaded")},
			wantReason: "ask claude for verdicts: 529 overloaded"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newHuntVerdictUnderTest(t, "BTC")
			underTest.everyCoinOnBinance()
			underTest.strategistAnswers(nil, testCase.answerErrors...)

			pipelineRun, synthesizeError := underTest.huntVerdictApplication.SynthesizeHuntVerdictsManually(context.Background())

			require.NoError(t, synthesizeError)
			assert.Equal(t, string(vo.PipelineRunStatusFailed), pipelineRun.Status)
			assert.Equal(t, testCase.wantReason, pipelineRun.FailureReason)
			assert.Nil(t, underTest.rewrittenBoard)
			assert.Nil(t, underTest.savedCoinVerdicts)
		})
	}
}

func TestSynthesizeHuntVerdictsAsksAgainOnceOnAnUnreadableAnswer(t *testing.T) {
	underTest := newHuntVerdictUnderTest(t, "BTC")
	underTest.everyCoinOnBinance()
	underTest.strategistAnswers([]vo.HuntVerdictAnswerVo{longAnswer("BTC")}, domains.ErrHuntVerdictAnswerUnusable, nil)

	pipelineRun, synthesizeError := underTest.huntVerdictApplication.SynthesizeHuntVerdictsManually(context.Background())

	require.NoError(t, synthesizeError)
	assert.Equal(t, string(vo.PipelineRunStatusSucceeded), pipelineRun.Status)
	assert.Equal(t, []string{"BTC"}, boardSymbols(underTest.rewrittenBoard))
}

func TestSynthesizeHuntVerdictsShowsTheFirstExchangeThatListsTheCoin(t *testing.T) {
	underTest := newHuntVerdictUnderTest(t, "STRK", "PONS")
	underTest.binanceMarket.EXPECT().FindMarketStructure(gomock.Any(), "STRK").Return(vo.PerpetualMarketStructureVo{}, false, errors.New("reset"))
	underTest.bybitMarket.EXPECT().FindMarketStructure(gomock.Any(), "STRK").Return(vo.PerpetualMarketStructureVo{ExchangeName: "Bybit", LastPrice: usd("0.2")}, true, nil)
	underTest.binanceMarket.EXPECT().FindMarketStructure(gomock.Any(), "PONS").Return(vo.PerpetualMarketStructureVo{}, false, nil)
	underTest.bybitMarket.EXPECT().FindMarketStructure(gomock.Any(), "PONS").Return(vo.PerpetualMarketStructureVo{}, false, nil)
	underTest.strategistAnswers([]vo.HuntVerdictAnswerVo{longAnswer("STRK"), longAnswer("PONS")})

	_, synthesizeError := underTest.huntVerdictApplication.SynthesizeHuntVerdictsManually(context.Background())

	require.NoError(t, synthesizeError)
	assert.Equal(t, "Bybit", underTest.shownMaterials[0].MarketStructure.ExchangeName)
	assert.Nil(t, underTest.shownMaterials[1].MarketStructure)
	assert.Equal(t, "0.18", text(underTest.rewrittenBoard[0].StopLossPrice))
	assert.Equal(t, "watch", underTest.rewrittenBoard[1].Action)
	assert.Equal(t, "查不到最新價格，無法設定停損", underTest.rewrittenBoard[1].Rationale)
}

func TestSynthesizeHuntVerdictsRefusesWithoutASuccessfulInsightRound(t *testing.T) {
	controller := gomock.NewController(t)
	pipelineRunRepository := mocks.NewMockIPipelineRunRepository(controller)
	pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepInsight)).Return(entities.PipelineRun{}, false, nil)
	huntVerdictApplication := application.NewHuntVerdictApplication(service.NewHuntVerdictService(
		pipelineRunRepository, nil, nil, nil, nil, nil, nil, huntVerdictPolicy()))

	_, synthesizeError := huntVerdictApplication.SynthesizeHuntVerdictsManually(context.Background())

	assert.EqualError(t, synthesizeError, "尚無成功的洞察輪次")
}

func TestSynthesizeHuntVerdictsSurfacesStorageFailures(t *testing.T) {
	t.Run("saving verdicts or rewriting the board fails the run and the call", func(t *testing.T) {
		for _, failing := range []string{"verdicts", "board"} {
			controller := gomock.NewController(t)
			underTest := newHuntVerdictUnderTest(t, "BTC")
			underTest.everyCoinOnBinance()
			underTest.strategistAnswers([]vo.HuntVerdictAnswerVo{longAnswer("BTC")})
			coinVerdicts := mocks.NewMockICoinVerdictRepository(controller)
			huntBoard := mocks.NewMockIHuntBoardRepository(controller)
			if failing == "verdicts" {
				coinVerdicts.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(errors.New("disk full"))
			} else {
				coinVerdicts.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil)
				huntBoard.EXPECT().Rewrite(gomock.Any(), gomock.Any()).Return(errors.New("rewrite hunt board: disk full"))
			}
			clockProxy := mocks.NewMockIClockProxy(controller)
			clockProxy.EXPECT().Now().Return(verdictStartedAt).AnyTimes()
			huntVerdictApplication := application.NewHuntVerdictApplication(service.NewHuntVerdictService(
				underTest.pipelineRunRepository, underTest.coinInsights, coinVerdicts, huntBoard,
				service.NewPerpetualMarketStructureService([]domaininterface.IPerpetualMarketStructureProxy{underTest.binanceMarket}, time.Second), underTest.strategist, clockProxy, huntVerdictPolicy()))

			_, synthesizeError := huntVerdictApplication.SynthesizeHuntVerdictsManually(context.Background())

			assert.ErrorContains(t, synthesizeError, "disk full", failing)
			assert.Equal(t, string(vo.PipelineRunStatusFailed), underTest.lastPipelineRunUpdate.Status, failing)
		}
	})

	testCases := []struct {
		name    string
		arrange func(pipelineRuns *mocks.MockIPipelineRunRepository, insights *mocks.MockICoinInsightRepository, strategist *mocks.MockIHuntVerdictStrategistProxy, verdicts *mocks.MockICoinVerdictRepository)
	}{
		{name: "finding the insight round", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, _ *mocks.MockICoinInsightRepository, _ *mocks.MockIHuntVerdictStrategistProxy, _ *mocks.MockICoinVerdictRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))
		}},
		{name: "finding the insights", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, insights *mocks.MockICoinInsightRepository, _ *mocks.MockIHuntVerdictStrategistProxy, _ *mocks.MockICoinVerdictRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 9}, true, nil)
			insights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, errors.New("disk"))
		}},
		{name: "recording the run", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, insights *mocks.MockICoinInsightRepository, _ *mocks.MockIHuntVerdictStrategistProxy, _ *mocks.MockICoinVerdictRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 9}, true, nil)
			insights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, errors.New("disk"))
		}},
		{name: "recording a strategist failure", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, insights *mocks.MockICoinInsightRepository, strategist *mocks.MockIHuntVerdictStrategistProxy, _ *mocks.MockICoinVerdictRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 9}, true, nil)
			insights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 13}, nil)
			strategist.EXPECT().SynthesizeVerdicts(gomock.Any(), gomock.Any()).Return(nil, errors.New("overloaded"))
			pipelineRuns.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("disk"))
		}},
		{name: "recording a storage failure", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, insights *mocks.MockICoinInsightRepository, strategist *mocks.MockIHuntVerdictStrategistProxy, verdicts *mocks.MockICoinVerdictRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 9}, true, nil)
			insights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 13}, nil)
			strategist.EXPECT().SynthesizeVerdicts(gomock.Any(), gomock.Any()).Return(nil, nil)
			verdicts.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(errors.New("disk"))
			pipelineRuns.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("disk"))
		}},
		{name: "recording the conclusion", arrange: func(pipelineRuns *mocks.MockIPipelineRunRepository, insights *mocks.MockICoinInsightRepository, strategist *mocks.MockIHuntVerdictStrategistProxy, verdicts *mocks.MockICoinVerdictRepository) {
			pipelineRuns.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 9}, true, nil)
			insights.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, nil)
			pipelineRuns.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 13}, nil)
			strategist.EXPECT().SynthesizeVerdicts(gomock.Any(), gomock.Any()).Return(nil, nil)
			verdicts.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil)
			pipelineRuns.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("disk"))
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			pipelineRuns := mocks.NewMockIPipelineRunRepository(controller)
			insights := mocks.NewMockICoinInsightRepository(controller)
			strategist := mocks.NewMockIHuntVerdictStrategistProxy(controller)
			verdicts := mocks.NewMockICoinVerdictRepository(controller)
			huntBoard := mocks.NewMockIHuntBoardRepository(controller)
			huntBoard.EXPECT().Rewrite(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			testCase.arrange(pipelineRuns, insights, strategist, verdicts)
			clockProxy := mocks.NewMockIClockProxy(controller)
			clockProxy.EXPECT().Now().Return(verdictStartedAt).AnyTimes()
			huntVerdictApplication := application.NewHuntVerdictApplication(service.NewHuntVerdictService(
				pipelineRuns, insights, verdicts, huntBoard, service.NewPerpetualMarketStructureService(nil, time.Second), strategist, clockProxy, huntVerdictPolicy()))

			_, synthesizeError := huntVerdictApplication.SynthesizeHuntVerdictsManually(context.Background())

			assert.ErrorContains(t, synthesizeError, "disk")
		})
	}
}

func TestGetHuntBoardAndVerdicts(t *testing.T) {
	t.Run("the board as stored, highest confidence first", func(t *testing.T) {
		underTest := newHuntVerdictUnderTest(t)
		underTest.huntBoard.EXPECT().FindAll(gomock.Any()).Return([]entities.HuntBoardEntry{
			{ID: 2, CoinSymbol: "PENGU", CalculatedAt: verdictStartedAt, PipelineRunID: 13, Action: "long", Confidence: 80, Leverage: 3, PositionSizeRatio: decimal.RequireFromString("0.05")},
			{ID: 1, CoinSymbol: "STRK", CalculatedAt: verdictStartedAt, PipelineRunID: 13, Action: "watch", Confidence: 40, PositionSizeRatio: decimal.Zero},
		}, nil)

		huntBoard, findError := underTest.huntVerdictApplication.GetHuntBoard(context.Background())

		require.NoError(t, findError)
		require.Len(t, huntBoard, 2)
		assert.Equal(t, dto.HuntBoardEntryDto{ID: 2, CoinSymbol: "PENGU", CalculatedAt: verdictStartedAt, CoinVerdictDto: dto.CoinVerdictDto{
			PipelineRunID: 13, CoinSymbol: "PENGU", Action: "long", Confidence: 80, Leverage: 3, PositionSizeRatio: decimal.RequireFromString("0.05")}}, huntBoard[0])
		assert.Equal(t, "STRK", huntBoard[1].CoinSymbol)
	})

	t.Run("the board when storage fails", func(t *testing.T) {
		underTest := newHuntVerdictUnderTest(t)
		underTest.huntBoard.EXPECT().FindAll(gomock.Any()).Return(nil, errors.New("disk"))

		_, findError := underTest.huntVerdictApplication.GetHuntBoard(context.Background())

		assert.ErrorContains(t, findError, "disk")
	})

	t.Run("a run's verdicts", func(t *testing.T) {
		underTest := newHuntVerdictUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(3)).Return(entities.PipelineRun{ID: 3}, nil)
		underTest.coinVerdicts.EXPECT().FindByPipelineRunID(gomock.Any(), uint(3)).Return([]entities.CoinVerdict{
			{PipelineRunID: 3, CoinSymbol: "PENGU", Action: "long", Confidence: 80}, {PipelineRunID: 3, CoinSymbol: "STRK", Action: "watch", Rationale: "CIO 未給出裁決"},
		}, nil)

		coinVerdicts, findError := underTest.huntVerdictApplication.GetCoinVerdictsOfPipelineRun(context.Background(), 3)

		require.NoError(t, findError)
		assert.Equal(t, []dto.CoinVerdictDto{{PipelineRunID: 3, CoinSymbol: "PENGU", Action: "long", Confidence: 80},
			{PipelineRunID: 3, CoinSymbol: "STRK", Action: "watch", Rationale: "CIO 未給出裁決"}}, coinVerdicts)
	})

	t.Run("an unknown run", func(t *testing.T) {
		underTest := newHuntVerdictUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(404)).Return(entities.PipelineRun{}, domains.ErrPipelineRunNotFound)

		_, findError := underTest.huntVerdictApplication.GetCoinVerdictsOfPipelineRun(context.Background(), 404)

		assert.ErrorIs(t, findError, domains.ErrPipelineRunNotFound)
	})

	t.Run("a run's verdicts when storage fails", func(t *testing.T) {
		underTest := newHuntVerdictUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(3)).Return(entities.PipelineRun{ID: 3}, nil)
		underTest.coinVerdicts.EXPECT().FindByPipelineRunID(gomock.Any(), uint(3)).Return(nil, errors.New("disk"))

		_, findError := underTest.huntVerdictApplication.GetCoinVerdictsOfPipelineRun(context.Background(), 3)

		assert.ErrorContains(t, findError, "disk")
	})
}
