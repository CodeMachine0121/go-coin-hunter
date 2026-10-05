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

// CoinFilteringService runs every filtering rule over the latest discovery's candidates.
type CoinFilteringService struct {
	pipelineRunRepository       domaininterface.IPipelineRunRepository
	coinCandidateRepository     domaininterface.ICoinCandidateRepository
	coinFilterResultRepository  domaininterface.ICoinFilterResultRepository
	coinProfileService          *CoinProfileService
	coinCandidateFilterHandlers []domaininterface.ICoinCandidateFilterHandler
	clockProxy                  domaininterface.IClockProxy
}

func NewCoinFilteringService(
	pipelineRunRepository domaininterface.IPipelineRunRepository,
	coinCandidateRepository domaininterface.ICoinCandidateRepository,
	coinFilterResultRepository domaininterface.ICoinFilterResultRepository,
	coinProfileService *CoinProfileService,
	coinCandidateFilterHandlers []domaininterface.ICoinCandidateFilterHandler,
	clockProxy domaininterface.IClockProxy,
) *CoinFilteringService {
	return &CoinFilteringService{
		pipelineRunRepository:       pipelineRunRepository,
		coinCandidateRepository:     coinCandidateRepository,
		coinFilterResultRepository:  coinFilterResultRepository,
		coinProfileService:          coinProfileService,
		coinCandidateFilterHandlers: coinCandidateFilterHandlers,
		clockProxy:                  clockProxy,
	}
}

// FilterCoinCandidates runs one filtering round over the newest successful discovery. Without one it refuses and
// records nothing; a data source failing as a whole fails the round and keeps no results.
func (coinFilteringService *CoinFilteringService) FilterCoinCandidates(
	executionContext context.Context, triggerSource vo.PipelineRunTriggerSourceVo,
) (dto.PipelineRunDto, error) {
	discoveryRun, discoveryFound, discoveryError := coinFilteringService.pipelineRunRepository.FindLatestSucceeded(
		executionContext, string(vo.PipelineRunStepDiscovery))
	if discoveryError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("find latest successful discovery run: %w", discoveryError)
	}
	if !discoveryFound {
		return dto.PipelineRunDto{}, domains.ErrNoSucceededDiscoveryRun
	}
	coinCandidates, candidatesError := coinFilteringService.coinCandidateRepository.FindByPipelineRunID(executionContext, discoveryRun.ID)
	if candidatesError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("find coin candidates to filter: %w", candidatesError)
	}

	startedAt := coinFilteringService.clockProxy.Now()
	pipelineRun, createError := coinFilteringService.pipelineRunRepository.Create(executionContext, entities.PipelineRun{
		Step:                     string(vo.PipelineRunStepFiltering),
		TriggerSource:            string(triggerSource),
		Status:                   string(vo.PipelineRunStatusRunning),
		TriggeredByPipelineRunID: &discoveryRun.ID,
		StartedAt:                startedAt,
	})
	if createError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("record filtering run: %w", createError)
	}

	keptCoinCount, filterError := func() (int, error) {
		coinProfiles, profileError := coinFilteringService.coinProfileService.AssembleCoinProfiles(executionContext, coinCandidates, startedAt)
		if profileError != nil {
			return 0, profileError
		}

		coinFilterResults := make([]entities.CoinFilterResult, 0, len(coinProfiles))
		keptCount := 0
		for _, coinProfile := range coinProfiles {
			filterVerdicts := make([]vo.FilterVerdictVo, 0, len(coinFilteringService.coinCandidateFilterHandlers))
			for _, coinCandidateFilterHandler := range coinFilteringService.coinCandidateFilterHandlers {
				filterVerdicts = append(filterVerdicts, coinCandidateFilterHandler.Evaluate(coinProfile))
			}
			coinFilterResult := domains.NewCoinFilterVerdictsDomain(filterVerdicts).ToCoinFilterResult(pipelineRun.ID, coinProfile.CoinSymbol)
			if coinFilterResult.IsKept {
				keptCount++
			}
			coinFilterResults = append(coinFilterResults, coinFilterResult)
		}
		if saveError := coinFilteringService.coinFilterResultRepository.CreateAll(executionContext, coinFilterResults); saveError != nil {
			return 0, fmt.Errorf("save coin filter results: %w", saveError)
		}

		return keptCount, nil
	}()
	if filterError != nil {
		failedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).Fail(filterError.Error(), coinFilteringService.clockProxy.Now())
		if updateError := coinFilteringService.pipelineRunRepository.Update(executionContext, failedPipelineRun); updateError != nil {
			return dto.PipelineRunDto{}, fmt.Errorf("record filtering run failure: %w (after %w)", updateError, filterError)
		}
		// An unavailable source is an expected outcome of a round, not a fault of the service.
		if errors.Is(filterError, domains.ErrCoinProfileSourceUnavailable) {
			return failedPipelineRun.ToDto(), nil
		}

		return dto.PipelineRunDto{}, filterError
	}

	concludedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).ConcludeFiltering(keptCoinCount, coinFilteringService.clockProxy.Now())
	if updateError := coinFilteringService.pipelineRunRepository.Update(executionContext, concludedPipelineRun); updateError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("record filtering run conclusion: %w", updateError)
	}

	return concludedPipelineRun.ToDto(), nil
}

// GetLatestKeptCoinCandidates returns the kept results of the newest successful filtering run, or none.
func (coinFilteringService *CoinFilteringService) GetLatestKeptCoinCandidates(
	executionContext context.Context,
) ([]dto.CoinFilterResultDto, error) {
	filteringRun, found, findError := coinFilteringService.pipelineRunRepository.FindLatestSucceeded(
		executionContext, string(vo.PipelineRunStepFiltering))
	if findError != nil {
		return nil, fmt.Errorf("find latest successful filtering run: %w", findError)
	}
	if !found {
		return []dto.CoinFilterResultDto{}, nil
	}

	coinFilterResults, resultsError := coinFilteringService.coinFilterResultRepository.FindByPipelineRunID(executionContext, filteringRun.ID)
	if resultsError != nil {
		return nil, fmt.Errorf("find coin filter results: %w", resultsError)
	}
	keptResultDtos := []dto.CoinFilterResultDto{}
	for _, coinFilterResult := range coinFilterResults {
		if coinFilterResult.IsKept {
			keptResultDtos = append(keptResultDtos, coinFilterResult.ToDto())
		}
	}

	return keptResultDtos, nil
}

// GetCoinFilterResultsOfPipelineRun returns every result of that run, kept or not; an unknown run is ErrPipelineRunNotFound.
func (coinFilteringService *CoinFilteringService) GetCoinFilterResultsOfPipelineRun(
	executionContext context.Context, pipelineRunID uint,
) ([]dto.CoinFilterResultDto, error) {
	if _, findError := coinFilteringService.pipelineRunRepository.FindOne(executionContext, pipelineRunID); findError != nil {
		return nil, findError
	}

	coinFilterResults, resultsError := coinFilteringService.coinFilterResultRepository.FindByPipelineRunID(executionContext, pipelineRunID)
	if resultsError != nil {
		return nil, fmt.Errorf("find coin filter results: %w", resultsError)
	}
	resultDtos := make([]dto.CoinFilterResultDto, 0, len(coinFilterResults))
	for _, coinFilterResult := range coinFilterResults {
		resultDtos = append(resultDtos, coinFilterResult.ToDto())
	}

	return resultDtos, nil
}
