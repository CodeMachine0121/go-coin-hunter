package service

import (
	"context"
	"errors"
	"fmt"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// HuntVerdictService asks the strategist for a verdict on every bullish coin at once and rewrites the hunt board with the longs.
type HuntVerdictService struct {
	pipelineRunRepository           domaininterface.IPipelineRunRepository
	coinInsightRepository           domaininterface.ICoinInsightRepository
	coinVerdictRepository           domaininterface.ICoinVerdictRepository
	huntBoardRepository             domaininterface.IHuntBoardRepository
	perpetualMarketStructureService *PerpetualMarketStructureService
	huntVerdictStrategistProxy      domaininterface.IHuntVerdictStrategistProxy
	clockProxy                      domaininterface.IClockProxy
	huntVerdictPolicy               vo.HuntVerdictPolicyVo
}

func NewHuntVerdictService(
	pipelineRunRepository domaininterface.IPipelineRunRepository,
	coinInsightRepository domaininterface.ICoinInsightRepository,
	coinVerdictRepository domaininterface.ICoinVerdictRepository,
	huntBoardRepository domaininterface.IHuntBoardRepository,
	perpetualMarketStructureService *PerpetualMarketStructureService,
	huntVerdictStrategistProxy domaininterface.IHuntVerdictStrategistProxy,
	clockProxy domaininterface.IClockProxy,
	huntVerdictPolicy vo.HuntVerdictPolicyVo,
) *HuntVerdictService {
	return &HuntVerdictService{
		pipelineRunRepository:           pipelineRunRepository,
		coinInsightRepository:           coinInsightRepository,
		coinVerdictRepository:           coinVerdictRepository,
		huntBoardRepository:             huntBoardRepository,
		perpetualMarketStructureService: perpetualMarketStructureService,
		huntVerdictStrategistProxy:      huntVerdictStrategistProxy,
		clockProxy:                      clockProxy,
		huntVerdictPolicy:               huntVerdictPolicy,
	}
}

// SynthesizeHuntVerdicts runs one verdict round over the bullish insights of the newest successful insight round. Without
// one it refuses and records nothing; without a bullish insight the strategist is not asked, the round is no data and the
// board is emptied; a strategist that cannot answer fails the round and leaves the hunt board as it was.
func (huntVerdictService *HuntVerdictService) SynthesizeHuntVerdicts(
	executionContext context.Context, triggerSource vo.PipelineRunTriggerSourceVo,
) (dto.PipelineRunDto, error) {
	insightRun, insightFound, insightRunError := huntVerdictService.pipelineRunRepository.FindLatestSucceeded(
		executionContext, string(vo.PipelineRunStepInsight))
	if insightRunError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("find latest successful insight run: %w", insightRunError)
	}
	if !insightFound {
		return dto.PipelineRunDto{}, domains.ErrNoSucceededInsightRun
	}
	coinInsights, insightsError := huntVerdictService.coinInsightRepository.FindByPipelineRunID(executionContext, insightRun.ID)
	if insightsError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("find coin insights to weigh: %w", insightsError)
	}

	startedAt := huntVerdictService.clockProxy.Now()
	pipelineRun, createError := huntVerdictService.pipelineRunRepository.Create(executionContext, entities.PipelineRun{
		Step:                     string(vo.PipelineRunStepVerdict),
		TriggerSource:            string(triggerSource),
		Status:                   string(vo.PipelineRunStatusRunning),
		TriggeredByPipelineRunID: &insightRun.ID,
		StartedAt:                startedAt,
	})
	if createError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("record verdict run: %w", createError)
	}

	// Every bullish coin is shown with the market as it is now, all coins asked for at once.
	bullishFocus := domains.NewBullishFocusDomain(huntVerdictService.huntVerdictPolicy)
	bullishInsights := bullishFocus.SelectBullishInsights(coinInsights)
	bullishCoinSymbols := make([]string, 0, len(bullishInsights))
	for _, bullishInsight := range bullishInsights {
		bullishCoinSymbols = append(bullishCoinSymbols, bullishInsight.CoinSymbol)
	}
	marketStructures := huntVerdictService.perpetualMarketStructureService.FindMarketStructures(executionContext, bullishCoinSymbols)
	materials := []vo.HuntVerdictMaterialVo{}
	for _, coinInsight := range bullishInsights {
		material := vo.HuntVerdictMaterialVo{CoinSymbol: coinInsight.CoinSymbol, Direction: coinInsight.Direction, Strength: coinInsight.Strength,
			Catalyst: coinInsight.Catalyst, Risks: coinInsight.Risks, Evidence: coinInsight.Evidence, DataGaps: coinInsight.DataGaps}
		if marketStructure, found := marketStructures[coinInsight.CoinSymbol]; found {
			material.MarketStructure = &marketStructure
		}
		materials = append(materials, material)
	}

	// Asking: an unreadable answer is asked again once; a strategist that still cannot answer fails the round, which is
	// an expected outcome rather than a fault, and the board is left as it was.
	coinVerdicts, synthesizeError := func() ([]entities.CoinVerdict, error) {
		if len(materials) == 0 {
			return []entities.CoinVerdict{}, nil
		}
		answers, answerError := huntVerdictService.huntVerdictStrategistProxy.SynthesizeVerdicts(executionContext, materials)
		if errors.Is(answerError, domains.ErrHuntVerdictAnswerUnusable) {
			answers, answerError = huntVerdictService.huntVerdictStrategistProxy.SynthesizeVerdicts(executionContext, materials)
		}
		if errors.Is(answerError, domains.ErrHuntVerdictAnswerUnusable) {
			return nil, domains.ErrHuntVerdictAnswerUnusable
		}
		if answerError != nil {
			return nil, answerError
		}

		return domains.NewHuntVerdictsDomain(materials, answers, huntVerdictService.huntVerdictPolicy).ToCoinVerdicts(pipelineRun.ID), nil
	}()
	if synthesizeError != nil {
		failedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).Fail(synthesizeError.Error(), huntVerdictService.clockProxy.Now())
		if updateError := huntVerdictService.pipelineRunRepository.Update(executionContext, failedPipelineRun); updateError != nil {
			return dto.PipelineRunDto{}, fmt.Errorf("record verdict run failure: %w (after %w)", updateError, synthesizeError)
		}

		return failedPipelineRun.ToDto(), nil
	}

	// Keeping: every verdict of the round becomes its history, then the board is rewritten in one go with the longs alone.
	storageError := func() error {
		if len(coinVerdicts) > 0 {
			if saveError := huntVerdictService.coinVerdictRepository.CreateAll(executionContext, coinVerdicts); saveError != nil {
				return fmt.Errorf("save coin verdicts: %w", saveError)
			}
		}

		return huntVerdictService.huntBoardRepository.Rewrite(executionContext, bullishFocus.ToHuntBoardEntries(coinVerdicts, startedAt))
	}()
	if storageError != nil {
		failedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).Fail(storageError.Error(), huntVerdictService.clockProxy.Now())
		if updateError := huntVerdictService.pipelineRunRepository.Update(executionContext, failedPipelineRun); updateError != nil {
			return dto.PipelineRunDto{}, fmt.Errorf("record verdict run failure: %w (after %w)", updateError, storageError)
		}

		return dto.PipelineRunDto{}, storageError
	}

	concludedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).ConcludeVerdict(len(materials), huntVerdictService.clockProxy.Now())
	if updateError := huntVerdictService.pipelineRunRepository.Update(executionContext, concludedPipelineRun); updateError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("record verdict run conclusion: %w", updateError)
	}

	return concludedPipelineRun.ToDto(), nil
}

// ClearHuntBoard empties the hunt board: a round that ran cleanly and found nothing worth going long leaves no stale longs.
func (huntVerdictService *HuntVerdictService) ClearHuntBoard(executionContext context.Context) error {
	return huntVerdictService.huntBoardRepository.Rewrite(executionContext, []entities.HuntBoardEntry{})
}

// GetHuntBoard returns the hunt board, highest confidence first.
func (huntVerdictService *HuntVerdictService) GetHuntBoard(executionContext context.Context) ([]dto.HuntBoardEntryDto, error) {
	huntBoardEntries, findError := huntVerdictService.huntBoardRepository.FindAll(executionContext)
	if findError != nil {
		return nil, fmt.Errorf("find hunt board: %w", findError)
	}
	huntBoardEntryDtos := make([]dto.HuntBoardEntryDto, 0, len(huntBoardEntries))
	for _, huntBoardEntry := range huntBoardEntries {
		huntBoardEntryDtos = append(huntBoardEntryDtos, huntBoardEntry.ToDto())
	}

	return huntBoardEntryDtos, nil
}

// GetCoinVerdictsOfPipelineRun returns every verdict of that run; an unknown run is ErrPipelineRunNotFound.
func (huntVerdictService *HuntVerdictService) GetCoinVerdictsOfPipelineRun(
	executionContext context.Context, pipelineRunID uint,
) ([]dto.CoinVerdictDto, error) {
	if _, findError := huntVerdictService.pipelineRunRepository.FindOne(executionContext, pipelineRunID); findError != nil {
		return nil, findError
	}

	coinVerdicts, verdictsError := huntVerdictService.coinVerdictRepository.FindByPipelineRunID(executionContext, pipelineRunID)
	if verdictsError != nil {
		return nil, fmt.Errorf("find coin verdicts: %w", verdictsError)
	}
	coinVerdictDtos := make([]dto.CoinVerdictDto, 0, len(coinVerdicts))
	for _, coinVerdict := range coinVerdicts {
		coinVerdictDtos = append(coinVerdictDtos, coinVerdict.ToDto())
	}

	return coinVerdictDtos, nil
}
