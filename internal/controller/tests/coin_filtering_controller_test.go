package controller_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

type filteringRoutesUnderTest struct {
	engine                     *gin.Engine
	pipelineRunRepository      *mocks.MockIPipelineRunRepository
	coinCandidateRepository    *mocks.MockICoinCandidateRepository
	coinFilterResultRepository *mocks.MockICoinFilterResultRepository
	coinIntelligenceRepository *mocks.MockICoinIntelligenceRepository
}

func newFilteringRoutesUnderTest(t *testing.T) filteringRoutesUnderTest {
	gin.SetMode(gin.TestMode)
	mockController := gomock.NewController(t)
	underTest := filteringRoutesUnderTest{
		engine:                     gin.New(),
		pipelineRunRepository:      mocks.NewMockIPipelineRunRepository(mockController),
		coinCandidateRepository:    mocks.NewMockICoinCandidateRepository(mockController),
		coinFilterResultRepository: mocks.NewMockICoinFilterResultRepository(mockController),
		coinIntelligenceRepository: mocks.NewMockICoinIntelligenceRepository(mockController),
	}
	clockProxy := mocks.NewMockIClockProxy(mockController)
	clockProxy.EXPECT().Now().Return(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)).AnyTimes()
	coinFilteringController := controller.NewCoinFilteringController(application.NewCoinFilteringApplication(service.NewCoinFilteringService(
		underTest.pipelineRunRepository, underTest.coinCandidateRepository, underTest.coinFilterResultRepository,
		service.NewCoinProfileService(underTest.coinIntelligenceRepository, nil, nil, nil, nil, time.Second), nil, clockProxy)))
	underTest.engine.POST("/coin-filterings", coinFilteringController.FilterCoinCandidates)
	underTest.engine.GET("/coin-filter-results/latest-kept", coinFilteringController.GetLatestKeptCoinCandidates)
	underTest.engine.GET("/pipeline-runs/:pipelineRunId/coin-filter-results", coinFilteringController.GetCoinFilterResultsOfPipelineRun)

	return underTest
}

func (underTest filteringRoutesUnderTest) request(method, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	underTest.engine.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
	return recorder
}

func errorMessageOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	body := map[string]string{}
	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body["error"]
}

func TestFilteringRoutes(t *testing.T) {
	t.Run("nothing to filter is a conflict", func(t *testing.T) {
		underTest := newFilteringRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepDiscovery)).Return(entities.PipelineRun{}, false, nil)

		recorder := underTest.request(http.MethodPost, "/coin-filterings")

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.Equal(t, "尚無成功的探索輪次", errorMessageOf(t, recorder))
	})

	t.Run("a finished round answers with its run", func(t *testing.T) {
		underTest := newFilteringRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 5}, true, nil)
		underTest.coinCandidateRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(5)).Return(nil, nil)
		underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, pipelineRun entities.PipelineRun) (entities.PipelineRun, error) {
				pipelineRun.ID = 9
				return pipelineRun, nil
			})
		underTest.coinIntelligenceRepository.EXPECT().FindDeclaredContractAddresses(gomock.Any(), gomock.Any()).Return(map[string]vo.TokenAddressVo{}, nil)
		underTest.coinFilterResultRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil)
		underTest.pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		recorder := underTest.request(http.MethodPost, "/coin-filterings")

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := map[string]any{}
		assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		assert.Equal(t, float64(9), body["id"])
		assert.Equal(t, float64(5), body["triggeredByPipelineRunId"])
		assert.Equal(t, "noData", body["status"])
	})

	t.Run("a storage fault is a server error", func(t *testing.T) {
		underTest := newFilteringRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodPost, "/coin-filterings").Code)
	})

	t.Run("latest kept candidates", func(t *testing.T) {
		underTest := newFilteringRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), string(vo.PipelineRunStepFiltering)).Return(entities.PipelineRun{}, false, nil)

		recorder := underTest.request(http.MethodGet, "/coin-filter-results/latest-kept")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `[]`, recorder.Body.String())
	})

	t.Run("latest kept candidates when storage fails", func(t *testing.T) {
		underTest := newFilteringRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodGet, "/coin-filter-results/latest-kept").Code)
	})
}

func TestFilterResultsOfRunRoute(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		arrange    func(underTest filteringRoutesUnderTest)
		wantStatus int
		wantError  string
	}{
		{name: "a non-numeric run", path: "/pipeline-runs/abc/coin-filter-results", arrange: func(filteringRoutesUnderTest) {},
			wantStatus: http.StatusBadRequest, wantError: "輪次編號必須是正整數"},
		{name: "an unknown run", path: "/pipeline-runs/9/coin-filter-results", arrange: func(underTest filteringRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{}, domains.ErrPipelineRunNotFound)
		}, wantStatus: http.StatusNotFound, wantError: "找不到這個輪次"},
		{name: "storage failing", path: "/pipeline-runs/9/coin-filter-results", arrange: func(underTest filteringRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{ID: 9}, nil)
			underTest.coinFilterResultRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, errors.New("disk"))
		}, wantStatus: http.StatusInternalServerError},
		{name: "a known run", path: "/pipeline-runs/9/coin-filter-results", arrange: func(underTest filteringRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{ID: 9}, nil)
			underTest.coinFilterResultRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return([]entities.CoinFilterResult{}, nil)
		}, wantStatus: http.StatusOK},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newFilteringRoutesUnderTest(t)
			testCase.arrange(underTest)

			recorder := underTest.request(http.MethodGet, testCase.path)

			assert.Equal(t, testCase.wantStatus, recorder.Code)
			if testCase.wantError != "" {
				assert.Equal(t, testCase.wantError, errorMessageOf(t, recorder))
			}
		})
	}
}
