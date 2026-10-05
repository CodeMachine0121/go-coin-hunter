package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
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
	pipelineRunID, parseError := strconv.ParseUint(context.Param("pipelineRunId"), 10, 64)
	if parseError != nil || pipelineRunID == 0 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "輪次編號必須是正整數"})
		return
	}

	coinInsights, findError := coinInsightController.coinInsightApplication.GetCoinInsightsOfPipelineRun(context.Request.Context(), uint(pipelineRunID))
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
