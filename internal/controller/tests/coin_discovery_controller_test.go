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

type discoveryRoutesUnderTest struct {
	engine                     *gin.Engine
	pipelineRunRepository      *mocks.MockIPipelineRunRepository
	coinIntelligenceRepository *mocks.MockICoinIntelligenceRepository
	coinCandidateRepository    *mocks.MockICoinCandidateRepository
	outcomeRepository          *mocks.MockIInformationSourceOutcomeRepository
}

func newDiscoveryRoutesUnderTest(t *testing.T) discoveryRoutesUnderTest {
	gin.SetMode(gin.TestMode)
	mockController := gomock.NewController(t)
	underTest := discoveryRoutesUnderTest{
		engine:                     gin.New(),
		pipelineRunRepository:      mocks.NewMockIPipelineRunRepository(mockController),
		coinIntelligenceRepository: mocks.NewMockICoinIntelligenceRepository(mockController),
		coinCandidateRepository:    mocks.NewMockICoinCandidateRepository(mockController),
		outcomeRepository:          mocks.NewMockIInformationSourceOutcomeRepository(mockController),
	}
	clockProxy := mocks.NewMockIClockProxy(mockController)
	clockProxy.EXPECT().Now().Return(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)).AnyTimes()

	coinDiscoveryController := controller.NewCoinDiscoveryController(application.NewCoinDiscoveryApplication(
		service.NewCoinDiscoveryService(nil, underTest.pipelineRunRepository, underTest.outcomeRepository,
			underTest.coinIntelligenceRepository, underTest.coinCandidateRepository, clockProxy, vo.DiscoveryPolicyVo{})))
	pipelineRunController := controller.NewPipelineRunController(application.NewPipelineRunApplication(
		service.NewPipelineRunService(underTest.pipelineRunRepository, clockProxy)))
	underTest.engine.POST("/coin-discoveries", coinDiscoveryController.DiscoverCoins)
	underTest.engine.GET("/coin-candidates/latest", coinDiscoveryController.GetLatestCoinCandidates)
	underTest.engine.GET("/pipeline-runs/:pipelineRunId/coin-intelligences", coinDiscoveryController.GetCoinIntelligencesOfPipelineRun)
	underTest.engine.GET("/pipeline-runs", pipelineRunController.GetPipelineRuns)

	return underTest
}

func (underTest discoveryRoutesUnderTest) request(method, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	underTest.engine.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
	return recorder
}

func TestCoinIntelligencesRouteStatus(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		arrange    func(underTest discoveryRoutesUnderTest)
		wantStatus int
	}{
		{name: "a non-numeric run", path: "/pipeline-runs/abc/coin-intelligences", arrange: func(discoveryRoutesUnderTest) {}, wantStatus: http.StatusBadRequest},
		{name: "run zero", path: "/pipeline-runs/0/coin-intelligences", arrange: func(discoveryRoutesUnderTest) {}, wantStatus: http.StatusBadRequest},
		{name: "an unknown run", path: "/pipeline-runs/9/coin-intelligences", arrange: func(underTest discoveryRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{}, domains.ErrPipelineRunNotFound)
		}, wantStatus: http.StatusNotFound},
		{name: "storage failing", path: "/pipeline-runs/9/coin-intelligences", arrange: func(underTest discoveryRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{ID: 9}, nil)
			underTest.coinIntelligenceRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return(nil, errors.New("disk"))
		}, wantStatus: http.StatusInternalServerError},
		{name: "a known run", path: "/pipeline-runs/9/coin-intelligences", arrange: func(underTest discoveryRoutesUnderTest) {
			underTest.pipelineRunRepository.EXPECT().FindOne(gomock.Any(), uint(9)).Return(entities.PipelineRun{ID: 9}, nil)
			underTest.coinIntelligenceRepository.EXPECT().FindByPipelineRunID(gomock.Any(), uint(9)).Return([]entities.CoinIntelligence{}, nil)
		}, wantStatus: http.StatusOK},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			underTest := newDiscoveryRoutesUnderTest(t)
			testCase.arrange(underTest)

			assert.Equal(t, testCase.wantStatus, underTest.request(http.MethodGet, testCase.path).Code)
		})
	}
}

func TestDiscoveryRoutesAnswer(t *testing.T) {
	t.Run("manual discovery answers with the run", func(t *testing.T) {
		underTest := newDiscoveryRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{ID: 4}, nil)
		underTest.pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		underTest.outcomeRepository.EXPECT().CreateAll(gomock.Any(), gomock.Any()).Return(nil)
		underTest.coinIntelligenceRepository.EXPECT().SaveNew(gomock.Any(), gomock.Any()).Return(nil)

		recorder := underTest.request(http.MethodPost, "/coin-discoveries")

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := map[string]any{}
		assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		assert.Equal(t, float64(4), body["id"])
		assert.Equal(t, "failed", body["status"])
	})

	t.Run("manual discovery that cannot be recorded is a server error", func(t *testing.T) {
		underTest := newDiscoveryRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().Create(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodPost, "/coin-discoveries").Code)
	})

	t.Run("latest candidates", func(t *testing.T) {
		underTest := newDiscoveryRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, nil)

		recorder := underTest.request(http.MethodGet, "/coin-candidates/latest")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `[]`, recorder.Body.String())
	})

	t.Run("latest candidates when storage fails", func(t *testing.T) {
		underTest := newDiscoveryRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindLatestSucceeded(gomock.Any(), gomock.Any()).Return(entities.PipelineRun{}, false, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodGet, "/coin-candidates/latest").Code)
	})

	t.Run("pipeline runs", func(t *testing.T) {
		underTest := newDiscoveryRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindAll(gomock.Any()).Return([]entities.PipelineRun{}, nil)

		assert.Equal(t, http.StatusOK, underTest.request(http.MethodGet, "/pipeline-runs").Code)
	})

	t.Run("pipeline runs when storage fails", func(t *testing.T) {
		underTest := newDiscoveryRoutesUnderTest(t)
		underTest.pipelineRunRepository.EXPECT().FindAll(gomock.Any()).Return(nil, errors.New("disk"))

		assert.Equal(t, http.StatusInternalServerError, underTest.request(http.MethodGet, "/pipeline-runs").Code)
	})
}
