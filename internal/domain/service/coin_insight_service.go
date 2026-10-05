package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// CoinInsightService asks the analyst for one insight per kept candidate of the latest filtering.
type CoinInsightService struct {
	pipelineRunRepository      domaininterface.IPipelineRunRepository
	coinCandidateRepository    domaininterface.ICoinCandidateRepository
	coinFilterResultRepository domaininterface.ICoinFilterResultRepository
	coinInsightRepository      domaininterface.ICoinInsightRepository
	coinInsightMaterialService *CoinInsightMaterialService
	coinInsightAnalystProxy    domaininterface.ICoinInsightAnalystProxy
	clockProxy                 domaininterface.IClockProxy
	coinInsightPolicy          vo.CoinInsightPolicyVo
}

func NewCoinInsightService(
	pipelineRunRepository domaininterface.IPipelineRunRepository,
	coinCandidateRepository domaininterface.ICoinCandidateRepository,
	coinFilterResultRepository domaininterface.ICoinFilterResultRepository,
	coinInsightRepository domaininterface.ICoinInsightRepository,
	coinInsightMaterialService *CoinInsightMaterialService,
	coinInsightAnalystProxy domaininterface.ICoinInsightAnalystProxy,
	clockProxy domaininterface.IClockProxy,
	coinInsightPolicy vo.CoinInsightPolicyVo,
) *CoinInsightService {
	return &CoinInsightService{
		pipelineRunRepository:      pipelineRunRepository,
		coinCandidateRepository:    coinCandidateRepository,
		coinFilterResultRepository: coinFilterResultRepository,
		coinInsightRepository:      coinInsightRepository,
		coinInsightMaterialService: coinInsightMaterialService,
		coinInsightAnalystProxy:    coinInsightAnalystProxy,
		clockProxy:                 clockProxy,
		coinInsightPolicy:          coinInsightPolicy,
	}
}

// AnalyzeCoinCandidates runs one insight round over the newest successful filtering. Without one it refuses and records
// nothing; one coin's failed analysis never stops the others.
func (coinInsightService *CoinInsightService) AnalyzeCoinCandidates(
	executionContext context.Context, triggerSource vo.PipelineRunTriggerSourceVo,
) (dto.PipelineRunDto, error) {
	filteringRun, filteringFound, filteringError := coinInsightService.pipelineRunRepository.FindLatestSucceeded(
		executionContext, string(vo.PipelineRunStepFiltering))
	if filteringError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("find latest successful filtering run: %w", filteringError)
	}
	if !filteringFound {
		return dto.PipelineRunDto{}, domains.ErrNoSucceededFilteringRun
	}
	coinFilterResults, resultsError := coinInsightService.coinFilterResultRepository.FindByPipelineRunID(executionContext, filteringRun.ID)
	if resultsError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("find coin filter results to analyze: %w", resultsError)
	}
	coinCandidates := []entities.CoinCandidate{}
	if filteringRun.TriggeredByPipelineRunID != nil {
		foundCandidates, candidatesError := coinInsightService.coinCandidateRepository.FindByPipelineRunID(executionContext, *filteringRun.TriggeredByPipelineRunID)
		if candidatesError != nil {
			return dto.PipelineRunDto{}, fmt.Errorf("find coin candidates to order: %w", candidatesError)
		}
		coinCandidates = foundCandidates
	}
	selectedResults := domains.NewInsightCandidateSelectionDomain(coinInsightService.coinInsightPolicy.MaximumCoinsPerRound).
		SelectKeptResults(coinFilterResults, coinCandidates)

	startedAt := coinInsightService.clockProxy.Now()
	pipelineRun, createError := coinInsightService.pipelineRunRepository.Create(executionContext, entities.PipelineRun{
		Step:                     string(vo.PipelineRunStepInsight),
		TriggerSource:            string(triggerSource),
		Status:                   string(vo.PipelineRunStatusRunning),
		TriggeredByPipelineRunID: &filteringRun.ID,
		StartedAt:                startedAt,
	})
	if createError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("record insight run: %w", createError)
	}

	succeededCount, insightError := func() (int, error) {
		materials, materialError := coinInsightService.coinInsightMaterialService.GatherCoinInsightMaterials(executionContext, selectedResults, startedAt)
		if materialError != nil {
			return 0, materialError
		}

		// At most a few analyses at once keeps the AI spend and rate predictable; each coin is asked again once on an unreadable answer.
		coinInsights := make([]entities.CoinInsight, len(materials))
		analysisSlots := make(chan struct{}, coinInsightService.coinInsightPolicy.MaximumConcurrentAnalyses)
		waitGroup := sync.WaitGroup{}
		for index, material := range materials {
			waitGroup.Go(func() {
				analysisSlots <- struct{}{}
				defer func() { <-analysisSlots }()

				answer, analyzeError := coinInsightService.coinInsightAnalystProxy.AnalyzeCoin(executionContext, material)
				if errors.Is(analyzeError, domains.ErrCoinInsightAnswerUnusable) {
					answer, analyzeError = coinInsightService.coinInsightAnalystProxy.AnalyzeCoin(executionContext, material)
				}
				if analyzeError != nil {
					failureReason := analyzeError.Error()
					if errors.Is(analyzeError, domains.ErrCoinInsightAnswerUnusable) {
						failureReason = domains.ErrCoinInsightAnswerUnusable.Error()
					}
					coinInsights[index] = entities.CoinInsight{PipelineRunID: pipelineRun.ID, CoinSymbol: material.CoinSymbol,
						FailureReason: failureReason, Risks: []string{}, Evidence: []string{}, DataGaps: material.DataGaps}
					return
				}
				coinInsights[index] = domains.NewCoinInsightAnswerDomain(answer, material).ToCoinInsight(pipelineRun.ID)
			})
		}
		waitGroup.Wait()

		succeeded := 0
		for _, coinInsight := range coinInsights {
			if coinInsight.Succeeded {
				succeeded++
			}
		}
		if saveError := coinInsightService.coinInsightRepository.CreateAll(executionContext, coinInsights); saveError != nil {
			return 0, fmt.Errorf("save coin insights: %w", saveError)
		}

		return succeeded, nil
	}()
	if insightError != nil {
		failedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).Fail(insightError.Error(), coinInsightService.clockProxy.Now())
		if updateError := coinInsightService.pipelineRunRepository.Update(executionContext, failedPipelineRun); updateError != nil {
			return dto.PipelineRunDto{}, fmt.Errorf("record insight run failure: %w (after %w)", updateError, insightError)
		}

		return dto.PipelineRunDto{}, insightError
	}

	concludedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).ConcludeInsight(succeededCount, coinInsightService.clockProxy.Now())
	if updateError := coinInsightService.pipelineRunRepository.Update(executionContext, concludedPipelineRun); updateError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("record insight run conclusion: %w", updateError)
	}

	return concludedPipelineRun.ToDto(), nil
}

// GetLatestCoinInsights returns every insight of the newest successful insight run, or none.
func (coinInsightService *CoinInsightService) GetLatestCoinInsights(executionContext context.Context) ([]dto.CoinInsightDto, error) {
	insightRun, found, findError := coinInsightService.pipelineRunRepository.FindLatestSucceeded(executionContext, string(vo.PipelineRunStepInsight))
	if findError != nil {
		return nil, fmt.Errorf("find latest successful insight run: %w", findError)
	}
	if !found {
		return []dto.CoinInsightDto{}, nil
	}

	coinInsights, insightsError := coinInsightService.coinInsightRepository.FindByPipelineRunID(executionContext, insightRun.ID)
	if insightsError != nil {
		return nil, fmt.Errorf("find coin insights: %w", insightsError)
	}
	coinInsightDtos := make([]dto.CoinInsightDto, 0, len(coinInsights))
	for _, coinInsight := range coinInsights {
		coinInsightDtos = append(coinInsightDtos, coinInsight.ToDto())
	}

	return coinInsightDtos, nil
}

// GetCoinInsightsOfPipelineRun returns every result of that run, failures included; an unknown run is ErrPipelineRunNotFound.
func (coinInsightService *CoinInsightService) GetCoinInsightsOfPipelineRun(
	executionContext context.Context, pipelineRunID uint,
) ([]dto.CoinInsightDto, error) {
	if _, findError := coinInsightService.pipelineRunRepository.FindOne(executionContext, pipelineRunID); findError != nil {
		return nil, findError
	}

	coinInsights, insightsError := coinInsightService.coinInsightRepository.FindByPipelineRunID(executionContext, pipelineRunID)
	if insightsError != nil {
		return nil, fmt.Errorf("find coin insights: %w", insightsError)
	}
	coinInsightDtos := make([]dto.CoinInsightDto, 0, len(coinInsights))
	for _, coinInsight := range coinInsights {
		coinInsightDtos = append(coinInsightDtos, coinInsight.ToDto())
	}

	return coinInsightDtos, nil
}
