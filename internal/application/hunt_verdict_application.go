package application

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
)

type HuntVerdictApplication struct {
	huntVerdictService *service.HuntVerdictService
}

func NewHuntVerdictApplication(huntVerdictService *service.HuntVerdictService) *HuntVerdictApplication {
	return &HuntVerdictApplication{huntVerdictService: huntVerdictService}
}

// SynthesizeHuntVerdictsManually is the trader asking for a verdict round right now.
func (huntVerdictApplication *HuntVerdictApplication) SynthesizeHuntVerdictsManually(executionContext context.Context) (dto.PipelineRunDto, error) {
	return huntVerdictApplication.huntVerdictService.SynthesizeHuntVerdicts(executionContext, vo.PipelineRunTriggerSourceManual)
}

func (huntVerdictApplication *HuntVerdictApplication) GetHuntBoard(executionContext context.Context) ([]dto.HuntBoardEntryDto, error) {
	return huntVerdictApplication.huntVerdictService.GetHuntBoard(executionContext)
}

func (huntVerdictApplication *HuntVerdictApplication) GetCoinVerdictsOfPipelineRun(
	executionContext context.Context, pipelineRunID uint,
) ([]dto.CoinVerdictDto, error) {
	return huntVerdictApplication.huntVerdictService.GetCoinVerdictsOfPipelineRun(executionContext, pipelineRunID)
}
