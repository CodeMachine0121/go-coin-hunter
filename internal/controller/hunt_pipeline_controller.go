package controller

import (
	"errors"
	"net/http"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/gin-gonic/gin"
)

type HuntPipelineController struct {
	huntPipelineApplication domaininterface.IHuntPipelineApplication
	// shutdownStarted is closed when the service begins shutting down, so a manual round starts no further step.
	shutdownStarted <-chan struct{}
}

func NewHuntPipelineController(huntPipelineApplication domaininterface.IHuntPipelineApplication, shutdownStarted <-chan struct{}) *HuntPipelineController {
	return &HuntPipelineController{huntPipelineApplication: huntPipelineApplication, shutdownStarted: shutdownStarted}
}

// RunHuntRound runs a whole round now and answers with every step's run, finished or stopped; a stopped round is still a
// successful request, while a round already running is a conflict.
func (huntPipelineController *HuntPipelineController) RunHuntRound(context *gin.Context) {
	huntRound, roundError := huntPipelineController.huntPipelineApplication.RunHuntRound(
		context.Request.Context(), huntPipelineController.shutdownStarted, vo.PipelineRunTriggerSourceManual)
	if errors.Is(roundError, domains.ErrHuntRoundAlreadyRunning) {
		context.JSON(http.StatusConflict, gin.H{"error": roundError.Error()})
		return
	}

	context.JSON(http.StatusOK, huntRound)
}
