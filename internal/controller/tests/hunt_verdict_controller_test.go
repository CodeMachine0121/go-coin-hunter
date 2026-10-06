package controller_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/controller"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type verdictRoutesUnderTest struct {
	filteringRoutesUnderTest
	coinInsightRepository *mocks.MockICoinInsightRepository
	coinVerdictRepository *mocks.MockICoinVerdictRepository
	huntBoardRepository   *mocks.MockIHuntBoardRepository
	strategist            *mocks.MockIHuntVerdictStrategistProxy
}

func newVerdictRoutesUnderTest(t *testing.T) verdictRoutesUnderTest {
	gin.SetMode(gin.TestMode)
	mockController := gomock.NewController(t)
	underTest := verdictRoutesUnderTest{
		filteringRoutesUnderTest: filteringRoutesUnderTest{engine: gin.New(), pipelineRunRepository: mocks.NewMockIPipelineRunRepository(mockController)},
		coinInsightRepository:    mocks.NewMockICoinInsightRepository(mockController),
		coinVerdictRepository:    mocks.NewMockICoinVerdictRepository(mockController),
		huntBoardRepository:      mocks.NewMockIHuntBoardRepository(mockController),
		strategist:               mocks.NewMockIHuntVerdictStrategistProxy(mockController),
	}
	clockProxy := mocks.NewMockIClockProxy(mockController)
	clockProxy.EXPECT().Now().Return(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)).AnyTimes()
	huntVerdictController := controller.NewHuntVerdictController(application.NewHuntVerdictApplication(service.NewHuntVerdictService(
		underTest.pipelineRunRepository, underTest.coinInsightRepository, underTest.coinVerdictRepository, underTest.huntBoardRepository,
		service.NewPerpetualMarketStructureService(nil, time.Second), underTest.strategist, clockProxy, vo.HuntVerdictPolicyVo{})))
	underTest.engine.POST("/hunt-verdicts", huntVerdictController.SynthesizeHuntVerdicts)
	underTest.engine.GET("/hunt-board", huntVerdictController.GetHuntBoard)
	underTest.engine.GET("/pipeline-runs/:pipelineRunId/coin-verdicts", huntVerdictController.GetCoinVerdictsOfPipelineRun)

	return underTest
}

func TestVerdictRoutes(t *testing.T) {
	t.Run("nothing to weigh is a conflict", func(t *testing.T) {
		underTest := newVerdictRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepInsight)).Return(entities.PipelineRun{}, false, nil)

		recorder := underTest.request(http.MethodPost, "/hunt-verdicts")

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.Equal(t, "尚無成功的洞察輪次", errorMessageOf(t, recorder))
	})

	t.Run("a finished round answers with its run", func(t *testing.T) {
		underTest := newVerdictRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 9}, true, nil)
		underTest.coinInsightRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return([]entities.CoinInsight{
			{CoinSymbol: "PENGU", Succeeded: true, Direction: "bullish", Strength: 7}}, nil)
		underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, pipelineRun entities.PipelineRun) (entities.PipelineRun, error) {
				pipelineRun.ID = 13
				return pipelineRun, nil
			})
		underTest.strategist.EXPECT().SynthesizeVerdicts(gomock.Any(), gomock.Any()).Return(nil, nil)
		underTest.coinVerdictRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil)
		underTest.huntBoardRepository.EXPECT().Rewrite(gomock.Any(), gomock.Any()).Return(nil)
		underTest.pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		recorder := underTest.request(http.MethodPost, "/hunt-verdicts")

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := map[string]any{}
		assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		assert.Equal(t, float64(13), body["id"])
		assert.Equal(t, "succeeded", body["status"])
	})

	t.Run("a storage fault is a server error", func(t *testing.T) {
		underTest := newVerdictRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodPost, "/hunt-verdicts").Code)
	})

	t.Run("the hunt board", func(t *testing.T) {
		underTest := newVerdictRoutesUnderTest(t)
		calculatedAt := time.Date(2026, 10, 5, 18, 0, 0, 0, time.UTC)
		underTest.huntBoardRepository.EXPECT().FindAll(gomock.Any()).Return([]entities.HuntBoardEntry{
			{ID: 7, CoinSymbol: "PENGU", CalculatedAt: calculatedAt, PipelineRunID: 13, Action: "long", Confidence: 80, Leverage: 3, PositionSizeRatio: decimal.RequireFromString("0.05")},
		}, nil)

		recorder := underTest.request(http.MethodGet, "/hunt-board")

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := []map[string]any{}
		assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		assert.Equal(t, float64(7), body[0]["id"])
		assert.Equal(t, "PENGU", body[0]["coinSymbol"])
		assert.Equal(t, "2026-10-05T18:00:00Z", body[0]["calculatedAt"])
		assert.Equal(t, "long", body[0]["action"])
		assert.Equal(t, "0.05", body[0]["positionSizeRatio"])
	})

	t.Run("the hunt board when storage fails", func(t *testing.T) {
		underTest := newVerdictRoutesUnderTest(t)
		underTest.huntBoardRepository.EXPECT().FindAll(gomock.Any()).Return(nil, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodGet, "/hunt-board").Code)
	})
}

func TestVerdictsOfRunRoute(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		arrange    func(underTest verdictRoutesUnderTest)
		wantStatus int
		wantError  string
		wantBody   string
	}{
		{name: "a non-numeric run", path: "/pipeline-runs/abc/coin-verdicts", arrange: func(verdictRoutesUnderTest) {},
			wantStatus: http.StatusBadRequest, wantError: "輪次編號必須是正整數"},
		{name: "an unknown run", path: "/pipeline-runs/9/coin-verdicts", arrange: func(underTest verdictRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{}, domains.ErrPipelineRunNotFound)
		}, wantStatus: http.StatusNotFound, wantError: "找不到這個輪次"},
		{name: "storage failing", path: "/pipeline-runs/9/coin-verdicts", arrange: func(underTest verdictRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{ID: 9}, nil)
			underTest.coinVerdictRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, errors.New("disk"))
		}, wantStatus: http.StatusInternalServerError},
		{name: "a run of another step has no verdicts", path: "/pipeline-runs/9/coin-verdicts", arrange: func(underTest verdictRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{ID: 9, Step: string(vo.PipelineRunStepInsight)}, nil)
			underTest.coinVerdictRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return([]entities.CoinVerdict{}, nil)
		}, wantStatus: http.StatusOK, wantBody: `[]`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newVerdictRoutesUnderTest(t)
			testCase.arrange(underTest)

			recorder := underTest.request(http.MethodGet, testCase.path)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
			if testCase.wantError != "" {
				assert.Equal(t, testCase.wantError, errorMessageOf(t, recorder))
			}
			if testCase.wantBody != "" {
				assert.JSONEq(t, testCase.wantBody, recorder.Body.String())
			}
		})
	}
}
