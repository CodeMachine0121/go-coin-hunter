package controller

import (
	"net/http"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/gin-gonic/gin"
)

type PipelineRunController struct {
	pipelineRunApplication *application.PipelineRunApplication
}

func NewPipelineRunController(pipelineRunApplication *application.PipelineRunApplication) *PipelineRunController {
	return &PipelineRunController{pipelineRunApplication: pipelineRunApplication}
}

func (pipelineRunController *PipelineRunController) GetPipelineRuns(context *gin.Context) {
	pipelineRuns, findError := pipelineRunController.pipelineRunApplication.GetPipelineRuns(context.Request.Context())
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, pipelineRuns)
}
