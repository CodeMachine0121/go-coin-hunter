package domains

import (
	"slices"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// CoinCandidateSelectionDomain decides which coins in the stored intelligence are new coins worth hunting this round.
type CoinCandidateSelectionDomain struct {
	discoveryPolicy vo.DiscoveryPolicyVo
}

func NewCoinCandidateSelectionDomain(discoveryPolicy vo.DiscoveryPolicyVo) CoinCandidateSelectionDomain {
	return CoinCandidateSelectionDomain{discoveryPolicy: discoveryPolicy}
}

// WindowStart is the earliest publish time still inside the discovery window; the boundary itself is inside.
func (coinCandidateSelectionDomain CoinCandidateSelectionDomain) WindowStart(startedAt time.Time) time.Time {
	return startedAt.Add(-coinCandidateSelectionDomain.discoveryPolicy.Window)
}

// SelectCandidates groups eligible intelligence by coin symbol, one candidate per coin, ordered by symbol.
func (coinCandidateSelectionDomain CoinCandidateSelectionDomain) SelectCandidates(
	pipelineRunID uint, startedAt time.Time, coinIntelligences []entities.CoinIntelligence,
) []entities.CoinCandidate {
	windowStart := coinCandidateSelectionDomain.WindowStart(startedAt)
	excludedCoinSymbols := map[string]bool{}
	for _, excludedCoinSymbol := range coinCandidateSelectionDomain.discoveryPolicy.ExcludedCoinSymbols {
		excludedCoinSymbols[strings.ToUpper(strings.TrimSpace(excludedCoinSymbol))] = true
	}

	candidatesBySymbol := map[string]*entities.CoinCandidate{}
	sourcesBySymbol := map[string]map[string]bool{}
	for _, coinIntelligence := range coinIntelligences {
		if coinIntelligence.CoinSymbol == "" || coinIntelligence.IsTraditionalAsset ||
			excludedCoinSymbols[coinIntelligence.CoinSymbol] || coinIntelligence.PublishedAt.Before(windowStart) {
			continue
		}

		coinCandidate, known := candidatesBySymbol[coinIntelligence.CoinSymbol]
		if !known {
			coinCandidate = &entities.CoinCandidate{
				PipelineRunID:       pipelineRunID,
				CoinSymbol:          coinIntelligence.CoinSymbol,
				EarliestMentionedAt: coinIntelligence.PublishedAt,
			}
			candidatesBySymbol[coinIntelligence.CoinSymbol] = coinCandidate
			sourcesBySymbol[coinIntelligence.CoinSymbol] = map[string]bool{}
		}
		coinCandidate.IntelligenceCount++
		sourcesBySymbol[coinIntelligence.CoinSymbol][coinIntelligence.SourceName] = true
		coinCandidate.SourceCount = len(sourcesBySymbol[coinIntelligence.CoinSymbol])
		if coinIntelligence.PublishedAt.Before(coinCandidate.EarliestMentionedAt) {
			coinCandidate.EarliestMentionedAt = coinIntelligence.PublishedAt
		}
	}

	coinCandidates := make([]entities.CoinCandidate, 0, len(candidatesBySymbol))
	for _, coinCandidate := range candidatesBySymbol {
		coinCandidates = append(coinCandidates, *coinCandidate)
	}
	slices.SortFunc(coinCandidates, func(left, right entities.CoinCandidate) int {
		return strings.Compare(left.CoinSymbol, right.CoinSymbol)
	})

	return coinCandidates
}
