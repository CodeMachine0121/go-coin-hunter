package controller

import (
	"errors"
	"net/http"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/utilities"
	"github.com/gin-gonic/gin"
)

type CoinInsightController struct {
	coinInsightApplication *application.CoinInsightApplication
}

func NewCoinInsightController(coinInsightApplication *application.CoinInsightApplication) *CoinInsightController {
	return &CoinInsightController{coinInsightApplication: coinInsightApplication}
}

// AnalyzeCoinCandidates answers with the finished run, whatever its status; nothing to analyze is a conflict.
func (coinInsightController *CoinInsightController) AnalyzeCoinCandidates(context *gin.Context) {
	pipelineRun, analyzeError := coinInsightController.coinInsightApplication.AnalyzeCoinCandidatesManually(context.Request.Context())
	if errors.Is(analyzeError, domains.ErrNoSucceededFilteringRun) {
		context.JSON(http.StatusConflict, gin.H{"error": analyzeError.Error()})
		return
	}
	if analyzeError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": analyzeError.Error()})
		return
	}

	context.JSON(http.StatusOK, pipelineRun)
}

func (coinInsightController *CoinInsightController) GetLatestCoinInsights(context *gin.Context) {
	coinInsights, findError := coinInsightController.coinInsightApplication.GetLatestCoinInsights(context.Request.Context())
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, coinInsights)
}

func (coinInsightController *CoinInsightController) GetCoinInsightsOfPipelineRun(context *gin.Context) {
	pipelineRunID, parseError := utilities.PipelineRunIDFrom(context)
	if parseError != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": parseError.Error()})
		return
	}

	coinInsights, findError := coinInsightController.coinInsightApplication.GetCoinInsightsOfPipelineRun(context.Request.Context(), pipelineRunID)
	if errors.Is(findError, domains.ErrPipelineRunNotFound) {
		context.JSON(http.StatusNotFound, gin.H{"error": findError.Error()})
		return
	}
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, coinInsights)
}
