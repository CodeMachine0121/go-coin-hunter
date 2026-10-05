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

// CoinDiscoveryService asks every information source for intelligence and turns it into this round's coin candidates.
type CoinDiscoveryService struct {
	informationSourceProxies           []domaininterface.IInformationSourceProxy
	pipelineRunRepository              domaininterface.IPipelineRunRepository
	informationSourceOutcomeRepository domaininterface.IInformationSourceOutcomeRepository
	coinIntelligenceRepository         domaininterface.ICoinIntelligenceRepository
	coinCandidateRepository            domaininterface.ICoinCandidateRepository
	clockProxy                         domaininterface.IClockProxy
	discoveryPolicy                    vo.DiscoveryPolicyVo
}

func NewCoinDiscoveryService(
	informationSourceProxies []domaininterface.IInformationSourceProxy,
	pipelineRunRepository domaininterface.IPipelineRunRepository,
	informationSourceOutcomeRepository domaininterface.IInformationSourceOutcomeRepository,
	coinIntelligenceRepository domaininterface.ICoinIntelligenceRepository,
	coinCandidateRepository domaininterface.ICoinCandidateRepository,
	clockProxy domaininterface.IClockProxy,
	discoveryPolicy vo.DiscoveryPolicyVo,
) *CoinDiscoveryService {
	return &CoinDiscoveryService{
		informationSourceProxies:           informationSourceProxies,
		pipelineRunRepository:              pipelineRunRepository,
		informationSourceOutcomeRepository: informationSourceOutcomeRepository,
		coinIntelligenceRepository:         coinIntelligenceRepository,
		coinCandidateRepository:            coinCandidateRepository,
		clockProxy:                         clockProxy,
		discoveryPolicy:                    discoveryPolicy,
	}
}

// DiscoverCoins runs one discovery round; a run that cannot be recorded fails the whole round.
func (coinDiscoveryService *CoinDiscoveryService) DiscoverCoins(
	executionContext context.Context, triggerSource vo.PipelineRunTriggerSourceVo,
) (dto.PipelineRunDto, error) {
	startedAt := coinDiscoveryService.clockProxy.Now()
	pipelineRun, createError := coinDiscoveryService.pipelineRunRepository.Create(executionContext, entities.PipelineRun{
		Step:          string(vo.PipelineRunStepDiscovery),
		TriggerSource: string(triggerSource),
		Status:        string(vo.PipelineRunStatusRunning),
		StartedAt:     startedAt,
	})
	if createError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("record discovery run: %w", createError)
	}

	// Every source is asked at once under its own timeout, so one slow or broken source never holds up or cancels the others.
	informationSourceResults := make([]vo.InformationSourceResultVo, len(coinDiscoveryService.informationSourceProxies))
	waitGroup := sync.WaitGroup{}
	for index, informationSourceProxy := range coinDiscoveryService.informationSourceProxies {
		waitGroup.Go(func() {
			sourceContext, cancelSource := context.WithTimeout(
				executionContext, coinDiscoveryService.discoveryPolicy.SourceRequestTimeout)
			defer cancelSource()

			informationItems, fetchError := informationSourceProxy.FetchInformationItems(
				sourceContext, coinDiscoveryService.discoveryPolicy.ItemLimitPerSource)
			informationSourceResults[index] = vo.InformationSourceResultVo{SourceName: informationSourceProxy.SourceName()}
			if errors.Is(fetchError, context.DeadlineExceeded) {
				informationSourceResults[index].FailureReason = domains.InformationSourceTimedOutReason
				return
			}
			if fetchError != nil {
				informationSourceResults[index].FailureReason = fetchError.Error()
				return
			}
			informationSourceResults[index].InformationItems = informationItems
		})
	}
	waitGroup.Wait()

	informationSourceResultsDomain := domains.NewInformationSourceResultsDomain(informationSourceResults)
	informationSourceOutcomes := informationSourceResultsDomain.InformationSourceOutcomes(pipelineRun.ID)
	coinIntelligences := informationSourceResultsDomain.CoinIntelligences(pipelineRun.ID, coinDiscoveryService.clockProxy.Now())

	coinCandidateSelection := domains.NewCoinCandidateSelectionDomain(coinDiscoveryService.discoveryPolicy)
	coinCandidates, recordError := func() ([]entities.CoinCandidate, error) {
		if outcomesError := coinDiscoveryService.informationSourceOutcomeRepository.CreateAll(
			executionContext, informationSourceOutcomes); outcomesError != nil {
			return nil, fmt.Errorf("record information source outcomes: %w", outcomesError)
		}
		if saveError := coinDiscoveryService.coinIntelligenceRepository.SaveNew(
			executionContext, coinIntelligences); saveError != nil {
			return nil, fmt.Errorf("save coin intelligences: %w", saveError)
		}
		// Intelligence left in the window by earlier rounds must not make a round with no answering source look productive.
		if !informationSourceResultsDomain.AnySourceSucceeded() {
			return []entities.CoinCandidate{}, nil
		}
		windowCoinIntelligences, windowError := coinDiscoveryService.coinIntelligenceRepository.FindPublishedSince(
			executionContext, coinCandidateSelection.WindowStart(startedAt))
		if windowError != nil {
			return nil, fmt.Errorf("find coin intelligences in the discovery window: %w", windowError)
		}
		selectedCoinCandidates := coinCandidateSelection.SelectCandidates(pipelineRun.ID, startedAt, windowCoinIntelligences)
		if candidatesError := coinDiscoveryService.coinCandidateRepository.CreateAll(
			executionContext, selectedCoinCandidates); candidatesError != nil {
			return nil, fmt.Errorf("save coin candidates: %w", candidatesError)
		}

		return selectedCoinCandidates, nil
	}()
	if recordError != nil {
		failedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).Fail(recordError.Error(), coinDiscoveryService.clockProxy.Now())
		if updateError := coinDiscoveryService.pipelineRunRepository.Update(executionContext, failedPipelineRun); updateError != nil {
			return dto.PipelineRunDto{}, fmt.Errorf("record discovery run failure: %w (after %w)", updateError, recordError)
		}

		return dto.PipelineRunDto{}, recordError
	}

	concludedPipelineRun := domains.NewPipelineRunDomain(pipelineRun).ConcludeDiscovery(
		informationSourceResultsDomain, len(coinCandidates), coinDiscoveryService.clockProxy.Now())
	if updateError := coinDiscoveryService.pipelineRunRepository.Update(executionContext, concludedPipelineRun); updateError != nil {
		return dto.PipelineRunDto{}, fmt.Errorf("record discovery run conclusion: %w", updateError)
	}
	concludedPipelineRun.InformationSourceOutcomes = informationSourceOutcomes

	return concludedPipelineRun.ToDto(), nil
}

// GetLatestCoinCandidates returns the candidates of the newest successful discovery run, or none if there never was one.
func (coinDiscoveryService *CoinDiscoveryService) GetLatestCoinCandidates(
	executionContext context.Context,
) ([]dto.CoinCandidateDto, error) {
	pipelineRun, found, findError := coinDiscoveryService.pipelineRunRepository.FindLatestSucceeded(
		executionContext, string(vo.PipelineRunStepDiscovery))
	if findError != nil {
		return nil, fmt.Errorf("find latest successful discovery run: %w", findError)
	}
	if !found {
		return []dto.CoinCandidateDto{}, nil
	}

	coinCandidates, candidatesError := coinDiscoveryService.coinCandidateRepository.FindByPipelineRunID(
		executionContext, pipelineRun.ID)
	if candidatesError != nil {
		return nil, fmt.Errorf("find coin candidates: %w", candidatesError)
	}

	coinCandidateDtos := make([]dto.CoinCandidateDto, 0, len(coinCandidates))
	for _, coinCandidate := range coinCandidates {
		coinCandidateDtos = append(coinCandidateDtos, coinCandidate.ToDto())
	}

	return coinCandidateDtos, nil
}

// GetCoinIntelligencesOfPipelineRun returns the intelligence first saved by that run; an unknown run is ErrPipelineRunNotFound.
func (coinDiscoveryService *CoinDiscoveryService) GetCoinIntelligencesOfPipelineRun(
	executionContext context.Context, pipelineRunID uint,
) ([]dto.CoinIntelligenceDto, error) {
	if _, findError := coinDiscoveryService.pipelineRunRepository.FindOne(executionContext, pipelineRunID); findError != nil {
		return nil, findError
	}

	coinIntelligences, intelligencesError := coinDiscoveryService.coinIntelligenceRepository.FindByPipelineRunID(
		executionContext, pipelineRunID)
	if intelligencesError != nil {
		return nil, fmt.Errorf("find coin intelligences: %w", intelligencesError)
	}

	coinIntelligenceDtos := make([]dto.CoinIntelligenceDto, 0, len(coinIntelligences))
	for _, coinIntelligence := range coinIntelligences {
		coinIntelligenceDtos = append(coinIntelligenceDtos, coinIntelligence.ToDto())
	}

	return coinIntelligenceDtos, nil
}
