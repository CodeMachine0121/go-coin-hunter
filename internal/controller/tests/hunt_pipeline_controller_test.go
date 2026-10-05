package controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/controller"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHuntRoundRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("runs a manual round that stops when the service shuts down", func(t *testing.T) {
		shutdownStarted := make(chan struct{})
		huntPipeline := mocks.NewMockIHuntPipelineApplication(gomock.NewController(t))
		huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), vo.PipelineRunTriggerSourceManual).DoAndReturn(
			func(_ context.Context, stopBetweenSteps <-chan struct{}, _ vo.PipelineRunTriggerSourceVo) (dto.HuntRoundDto, error) {
				assert.Equal(t, (<-chan struct{})(shutdownStarted), stopBetweenSteps)
				return dto.HuntRoundDto{Steps: []dto.PipelineRunDto{{ID: 1, Step: "discovery", Status: "noData"}}, StoppedStep: "discovery", StoppedReason: "探索未成功：noData"}, nil
			})
		engine := gin.New()
		engine.POST("/hunt-rounds", controller.NewHuntPipelineController(huntPipeline, shutdownStarted).RunHuntRound)
		recorder := httptest.NewRecorder()

		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/hunt-rounds", nil))

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{"steps":[{"id":1,"step":"discovery","triggerSource":"","status":"noData","failureReason":"","triggeredByPipelineRunId":null,
			"startedAt":"0001-01-01T00:00:00Z","finishedAt":null,"informationSourceOutcomes":null}],"completed":false,"stoppedStep":"discovery","stoppedReason":"探索未成功：noData"}`,
			recorder.Body.String())
	})

	t.Run("a round already running is a conflict", func(t *testing.T) {
		huntPipeline := mocks.NewMockIHuntPipelineApplication(gomock.NewController(t))
		huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), gomock.Any()).Return(dto.HuntRoundDto{}, domains.ErrHuntRoundAlreadyRunning)
		engine := gin.New()
		engine.POST("/hunt-rounds", controller.NewHuntPipelineController(huntPipeline, nil).RunHuntRound)
		recorder := httptest.NewRecorder()

		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/hunt-rounds", nil))

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.Equal(t, "已有獵捕回合進行中", errorMessageOf(t, recorder))
	})
}
