package controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/controller"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHuntRoundRouteRunsAManualRound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	huntPipeline := mocks.NewMockIHuntPipelineApplication(gomock.NewController(t))
	huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Nil(), vo.PipelineRunTriggerSourceManual).DoAndReturn(
		func(context.Context, <-chan struct{}, vo.PipelineRunTriggerSourceVo) dto.HuntRoundDto {
			return dto.HuntRoundDto{Steps: []dto.PipelineRunDto{{ID: 1, Step: "discovery", Status: "noData"}}, StoppedStep: "discovery", StoppedReason: "探索未成功：noData"}
		})
	engine := gin.New()
	engine.POST("/hunt-rounds", controller.NewHuntPipelineController(huntPipeline).RunHuntRound)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/hunt-rounds", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"steps":[{"id":1,"step":"discovery","triggerSource":"","status":"noData","failureReason":"","triggeredByPipelineRunId":null,
		"startedAt":"0001-01-01T00:00:00Z","finishedAt":null,"informationSourceOutcomes":null}],"completed":false,"stoppedStep":"discovery","stoppedReason":"探索未成功：noData"}`,
		recorder.Body.String())
}
