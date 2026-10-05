package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/gin-gonic/gin"
)

type CoinFilteringController struct {
	coinFilteringApplication *application.CoinFilteringApplication
}

func NewCoinFilteringController(coinFilteringApplication *application.CoinFilteringApplication) *CoinFilteringController {
	return &CoinFilteringController{coinFilteringApplication: coinFilteringApplication}
}

// FilterCoinCandidates answers with the finished run, whatever its status; nothing to filter is a conflict.
func (coinFilteringController *CoinFilteringController) FilterCoinCandidates(context *gin.Context) {
	pipelineRun, filterError := coinFilteringController.coinFilteringApplication.FilterCoinCandidatesManually(context.Request.Context())
	if errors.Is(filterError, domains.ErrNoSucceededDiscoveryRun) {
		context.JSON(http.StatusConflict, gin.H{"error": filterError.Error()})
		return
	}
	if filterError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": filterError.Error()})
		return
	}

	context.JSON(http.StatusOK, pipelineRun)
}

func (coinFilteringController *CoinFilteringController) GetLatestKeptCoinCandidates(context *gin.Context) {
	coinFilterResults, findError := coinFilteringController.coinFilteringApplication.GetLatestKeptCoinCandidates(context.Request.Context())
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, coinFilterResults)
}

func (coinFilteringController *CoinFilteringController) GetCoinFilterResultsOfPipelineRun(context *gin.Context) {
	pipelineRunID, parseError := strconv.ParseUint(context.Param("pipelineRunId"), 10, 64)
	if parseError != nil || pipelineRunID == 0 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "輪次編號必須是正整數"})
		return
	}

	coinFilterResults, findError := coinFilteringController.coinFilteringApplication.GetCoinFilterResultsOfPipelineRun(
		context.Request.Context(), uint(pipelineRunID))
	if errors.Is(findError, domains.ErrPipelineRunNotFound) {
		context.JSON(http.StatusNotFound, gin.H{"error": findError.Error()})
		return
	}
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, coinFilterResults)
}
