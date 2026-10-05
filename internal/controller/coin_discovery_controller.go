package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/gin-gonic/gin"
)

type CoinDiscoveryController struct {
	coinDiscoveryApplication *application.CoinDiscoveryApplication
}

func NewCoinDiscoveryController(coinDiscoveryApplication *application.CoinDiscoveryApplication) *CoinDiscoveryController {
	return &CoinDiscoveryController{coinDiscoveryApplication: coinDiscoveryApplication}
}

// DiscoverCoins answers with the finished run, whatever its status; only an unrecordable run is a server error.
func (coinDiscoveryController *CoinDiscoveryController) DiscoverCoins(context *gin.Context) {
	pipelineRun, discoverError := coinDiscoveryController.coinDiscoveryApplication.DiscoverCoinsManually(context.Request.Context())
	if discoverError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": discoverError.Error()})
		return
	}

	context.JSON(http.StatusOK, pipelineRun)
}

func (coinDiscoveryController *CoinDiscoveryController) GetLatestCoinCandidates(context *gin.Context) {
	coinCandidates, findError := coinDiscoveryController.coinDiscoveryApplication.GetLatestCoinCandidates(context.Request.Context())
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, coinCandidates)
}

func (coinDiscoveryController *CoinDiscoveryController) GetCoinIntelligencesOfPipelineRun(context *gin.Context) {
	pipelineRunID, parseError := strconv.ParseUint(context.Param("pipelineRunId"), 10, 64)
	if parseError != nil || pipelineRunID == 0 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "輪次編號必須是正整數"})
		return
	}

	coinIntelligences, findError := coinDiscoveryController.coinDiscoveryApplication.GetCoinIntelligencesOfPipelineRun(
		context.Request.Context(), uint(pipelineRunID))
	if errors.Is(findError, domains.ErrPipelineRunNotFound) {
		context.JSON(http.StatusNotFound, gin.H{"error": findError.Error()})
		return
	}
	if findError != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": findError.Error()})
		return
	}

	context.JSON(http.StatusOK, coinIntelligences)
}
