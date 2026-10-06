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
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type insightRoutesUnderTest struct {
	filteringRoutesUnderTest
	coinInsightRepository *mocks.MockICoinInsightRepository
}

func newInsightRoutesUnderTest(t *testing.T) insightRoutesUnderTest {
	gin.SetMode(gin.TestMode)
	mockController := gomock.NewController(t)
	underTest := insightRoutesUnderTest{
		filteringRoutesUnderTest: filteringRoutesUnderTest{
			engine:                     gin.New(),
			pipelineRunRepository:      mocks.NewMockIPipelineRunRepository(mockController),
			coinCandidateRepository:    mocks.NewMockICoinCandidateRepository(mockController),
			coinFilterResultRepository: mocks.NewMockICoinFilterResultRepository(mockController),
			coinIntelligenceRepository: mocks.NewMockICoinIntelligenceRepository(mockController),
		},
		coinInsightRepository: mocks.NewMockICoinInsightRepository(mockController),
	}
	clockProxy := mocks.NewMockIClockProxy(mockController)
	clockProxy.EXPECT().Now().Return(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)).AnyTimes()
	policy := vo.CoinInsightPolicyVo{MaximumCoinsPerRound: 20, MaximumConcurrentAnalyses: 3}
	coinInsightController := controller.NewCoinInsightController(application.NewCoinInsightApplication(service.NewCoinInsightService(
		underTest.pipelineRunRepository, underTest.coinCandidateRepository, underTest.coinFilterResultRepository, underTest.coinInsightRepository,
		service.NewCoinInsightMaterialService(underTest.coinIntelligenceRepository, nil, service.NewPerpetualMarketStructureService(nil, time.Second, 5), policy), nil, clockProxy, policy)))
	underTest.engine.POST("/coin-insights", coinInsightController.AnalyzeCoinCandidates)
	underTest.engine.GET("/coin-insights/latest", coinInsightController.GetLatestCoinInsights)
	underTest.engine.GET("/pipeline-runs/:pipelineRunId/coin-insights", coinInsightController.GetCoinInsightsOfPipelineRun)

	return underTest
}

func TestInsightRoutes(t *testing.T) {
	t.Run("nothing to analyze is a conflict", func(t *testing.T) {
		underTest := newInsightRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepFiltering)).Return(entities.PipelineRun{}, false, nil)

		recorder := underTest.request(http.MethodPost, "/coin-insights")

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.Equal(t, "尚無成功的過濾輪次", errorMessageOf(t, recorder))
	})

	t.Run("a finished round answers with its run", func(t *testing.T) {
		underTest := newInsightRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 7}, true, nil)
		underTest.coinFilterResultRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(7)).Return(nil, nil)
		underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, pipelineRun entities.PipelineRun) (entities.PipelineRun, error) {
				pipelineRun.ID = 11
				return pipelineRun, nil
			})
		underTest.coinIntelligenceRepository.EXPECT().FindByCoinSymbolsSince(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
		underTest.coinInsightRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil)
		underTest.pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		recorder := underTest.request(http.MethodPost, "/coin-insights")

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := map[string]any{}
		assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		assert.Equal(t, float64(11), body["id"])
		assert.Equal(t, "failed", body["status"])
	})

	t.Run("a storage fault is a server error", func(t *testing.T) {
		underTest := newInsightRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodPost, "/coin-insights").Code)
	})

	t.Run("latest insights", func(t *testing.T) {
		underTest := newInsightRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepInsight)).Return(entities.PipelineRun{}, false, nil)

		recorder := underTest.request(http.MethodGet, "/coin-insights/latest")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `[]`, recorder.Body.String())
	})

	t.Run("latest insights when storage fails", func(t *testing.T) {
		underTest := newInsightRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodGet, "/coin-insights/latest").Code)
	})
}

func TestInsightsOfRunRoute(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		arrange    func(underTest insightRoutesUnderTest)
		wantStatus int
		wantError  string
		wantBody   string
	}{
		{name: "a non-numeric run", path: "/pipeline-runs/abc/coin-insights", arrange: func(insightRoutesUnderTest) {},
			wantStatus: http.StatusBadRequest, wantError: "輪次編號必須是正整數"},
		{name: "an unknown run", path: "/pipeline-runs/9/coin-insights", arrange: func(underTest insightRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{}, domains.ErrPipelineRunNotFound)
		}, wantStatus: http.StatusNotFound, wantError: "找不到這個輪次"},
		{name: "storage failing", path: "/pipeline-runs/9/coin-insights", arrange: func(underTest insightRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{ID: 9}, nil)
			underTest.coinInsightRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, errors.New("disk"))
		}, wantStatus: http.StatusInternalServerError},
		{name: "a run of another step has no insights", path: "/pipeline-runs/9/coin-insights", arrange: func(underTest insightRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{ID: 9, Step: string(vo.PipelineRunStepFiltering)}, nil)
			underTest.coinInsightRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return([]entities.CoinInsight{}, nil)
		}, wantStatus: http.StatusOK, wantBody: `[]`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newInsightRoutesUnderTest(t)
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
