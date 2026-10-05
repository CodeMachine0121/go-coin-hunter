package application

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
)

// HuntPipelineApplication runs discovery, filtering, insight and verdict in order. Each step reads the newest successful
// run of the step before it, so a step that does not succeed stops the round: going on would rework an older result.
type HuntPipelineApplication struct {
	// roundRunning is shared by every caller, scheduled or manual, so that at most one round runs at a time.
	roundRunning         atomic.Bool
	coinDiscoveryService *service.CoinDiscoveryService
	coinFilteringService *service.CoinFilteringService
	coinInsightService   *service.CoinInsightService
	huntVerdictService   *service.HuntVerdictService
}

func NewHuntPipelineApplication(
	coinDiscoveryService *service.CoinDiscoveryService,
	coinFilteringService *service.CoinFilteringService,
	coinInsightService *service.CoinInsightService,
	huntVerdictService *service.HuntVerdictService,
) *HuntPipelineApplication {
	return &HuntPipelineApplication{
		coinDiscoveryService: coinDiscoveryService,
		coinFilteringService: coinFilteringService,
		coinInsightService:   coinInsightService,
		huntVerdictService:   huntVerdictService,
	}
}

// RunHuntRound stops at the first step that errs or does not succeed, and starts no step once asked to stop or once the
// context has ended. Another round already running is refused; every round that runs leaves one log line.
func (huntPipelineApplication *HuntPipelineApplication) RunHuntRound(
	executionContext context.Context, stopBetweenSteps <-chan struct{}, triggerSource vo.PipelineRunTriggerSourceVo,
) (dto.HuntRoundDto, error) {
	if !huntPipelineApplication.roundRunning.CompareAndSwap(false, true) {
		return dto.HuntRoundDto{}, domains.ErrHuntRoundAlreadyRunning
	}
	defer huntPipelineApplication.roundRunning.Store(false)

	huntRound := huntPipelineApplication.runSteps(executionContext, stopBetweenSteps, triggerSource)
	pipelineRunIDs := make([]uint, 0, len(huntRound.Steps))
	for _, step := range huntRound.Steps {
		pipelineRunIDs = append(pipelineRunIDs, step.ID)
	}
	if huntRound.Completed {
		log.Printf("hunt round (%s) completed: pipeline runs %v", triggerSource, pipelineRunIDs)
	} else {
		log.Printf("hunt round (%s) stopped at %s (%s): pipeline runs %v", triggerSource, huntRound.StoppedStep, huntRound.StoppedReason, pipelineRunIDs)
	}

	return huntRound, nil
}

// runSteps runs the steps in order; it is apart from RunHuntRound only so the overlap flag is held around it by defer.
func (huntPipelineApplication *HuntPipelineApplication) runSteps(
	executionContext context.Context, stopBetweenSteps <-chan struct{}, triggerSource vo.PipelineRunTriggerSourceVo,
) dto.HuntRoundDto {
	steps := []struct {
		step vo.PipelineRunStepVo
		run  func(context.Context, vo.PipelineRunTriggerSourceVo) (dto.PipelineRunDto, error)
	}{
		{step: vo.PipelineRunStepDiscovery, run: huntPipelineApplication.coinDiscoveryService.DiscoverCoins},
		{step: vo.PipelineRunStepFiltering, run: huntPipelineApplication.coinFilteringService.FilterCoinCandidates},
		{step: vo.PipelineRunStepInsight, run: huntPipelineApplication.coinInsightService.AnalyzeCoinCandidates},
		{step: vo.PipelineRunStepVerdict, run: huntPipelineApplication.huntVerdictService.SynthesizeHuntVerdicts},
	}

	huntRound := dto.HuntRoundDto{Steps: []dto.PipelineRunDto{}}
	for _, pipelineStep := range steps {
		stepDomain := domains.NewPipelineRunStepDomain(pipelineStep.step)
		select {
		case <-stopBetweenSteps:
			huntRound.StoppedStep, huntRound.StoppedReason = string(pipelineStep.step), stepDomain.NotStartedReason("服務關閉中")
			return huntRound
		default:
		}
		if contextError := executionContext.Err(); contextError != nil {
			huntRound.StoppedStep, huntRound.StoppedReason = string(pipelineStep.step), stepDomain.NotStartedReason(contextError.Error())
			return huntRound
		}

		pipelineRun, runError := pipelineStep.run(executionContext, triggerSource)
		if runError != nil {
			huntRound.StoppedStep, huntRound.StoppedReason = string(pipelineStep.step), runError.Error()
			return huntRound
		}
		huntRound.Steps = append(huntRound.Steps, pipelineRun)
		if pipelineRun.Status != string(vo.PipelineRunStatusSucceeded) {
			huntRound.StoppedStep, huntRound.StoppedReason = string(pipelineStep.step), stepDomain.NotSucceededReason(pipelineRun.Status)
			return huntRound
		}
	}
	huntRound.Completed = true

	return huntRound
}
