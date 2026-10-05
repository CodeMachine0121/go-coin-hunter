package controller

import (
	"net/http"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/gin-gonic/gin"
)

type HuntPipelineController struct {
	huntPipelineApplication domaininterface.IHuntPipelineApplication
}

func NewHuntPipelineController(huntPipelineApplication domaininterface.IHuntPipelineApplication) *HuntPipelineController {
	return &HuntPipelineController{huntPipelineApplication: huntPipelineApplication}
}

// RunHuntRound runs a whole round now and answers with every step's run, finished or stopped; a stopped round is
// still a successful request.
func (huntPipelineController *HuntPipelineController) RunHuntRound(context *gin.Context) {
	context.JSON(http.StatusOK, huntPipelineController.huntPipelineApplication.RunHuntRound(
		context.Request.Context(), nil, vo.PipelineRunTriggerSourceManual))
}
