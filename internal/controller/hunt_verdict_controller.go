package controller

import (
	"errors"
	"net/http"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"github.com/gin-gonic/gin"
)

type HuntVerdictController struct {
	huntVerdictApplication *application.HuntVerdictApplication
}

func NewHuntVerdictController(huntVerdictApplication *application.HuntVerdictApplication) *HuntVerdictController {
	return &HuntVerdictController{huntVerdictApplication: huntVerdictApplication}
}

// SynthesizeHuntVerdicts answers with the finished run, whatever its status; nothing to weigh is a conflict.
func (huntVerdictController *HuntVerdictController) SynthesizeHuntVerdicts(context *gin.Context) {
	pipelineRun, synthesizeError := huntVerdictController.huntVerdictApplication.SynthesizeHuntVerdictsManually(context.Request.Context())
	if errors.Is(synthesizeError, domains.ErrNoSucceededInsightRun) {
		context.JSON(http.StatusConflict, gin.H{"error": synthesizeError.Error()})
		return
	}
	if synthesizeError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": synthesizeError.Error()})
		return
	}

	context.JSON(http.StatusOK, pipelineRun)
}

func (huntVerdictController *HuntVerdictController) GetHuntBoard(context *gin.Context) {
	huntBoard, findError := huntVerdictController.huntVerdictApplication.GetHuntBoard(context.Request.Context())
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, huntBoard)
}

func (huntVerdictController *HuntVerdictController) GetCoinVerdictsOfPipelineRun(context *gin.Context) {
	pipelineRunID, parseError := utilities.PipelineRunIDFrom(context)
	if parseError != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": parseError.Error()})
		return
	}

	coinVerdicts, findError := huntVerdictController.huntVerdictApplication.GetCoinVerdictsOfPipelineRun(context.Request.Context(), pipelineRunID)
	if errors.Is(findError, domains.ErrPipelineRunNotFound) {
		context.JSON(http.StatusNotFound, gin.H{"error": findError.Error()})
		return
	}
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, coinVerdicts)
}
