package domains

import (
	"slices"
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
)

// InsightCandidateSelectionDomain picks which kept candidates a round can afford to analyze.
type InsightCandidateSelectionDomain struct {
	maximumCoinsPerRound int
}

func NewInsightCandidateSelectionDomain(maximumCoinsPerRound int) InsightCandidateSelectionDomain {
	return InsightCandidateSelectionDomain{maximumCoinsPerRound: maximumCoinsPerRound}
}

// SelectKeptResults keeps the kept results only, earliest first mentioned first (symbol breaks ties), up to the cap.
func (insightCandidateSelectionDomain InsightCandidateSelectionDomain) SelectKeptResults(
	coinFilterResults []entities.CoinFilterResult, coinCandidates []entities.CoinCandidate,
) []entities.CoinFilterResult {
	coinCandidateBySymbol := map[string]entities.CoinCandidate{}
	for _, coinCandidate := range coinCandidates {
		coinCandidateBySymbol[coinCandidate.CoinSymbol] = coinCandidate
	}

	keptResults := []entities.CoinFilterResult{}
	for _, coinFilterResult := range coinFilterResults {
		if coinFilterResult.IsKept {
			keptResults = append(keptResults, coinFilterResult)
		}
	}
	// A kept coin with no candidate record has no first mention and goes after every coin that has one.
	slices.SortStableFunc(keptResults, func(left, right entities.CoinFilterResult) int {
		leftCandidate, leftKnown := coinCandidateBySymbol[left.CoinSymbol]
		rightCandidate, rightKnown := coinCandidateBySymbol[right.CoinSymbol]
		if leftKnown != rightKnown {
			if leftKnown {
				return -1
			}
			return 1
		}
		if comparison := leftCandidate.EarliestMentionedAt.Compare(rightCandidate.EarliestMentionedAt); comparison != 0 {
			return comparison
		}
		return strings.Compare(left.CoinSymbol, right.CoinSymbol)
	})

	return keptResults[:min(insightCandidateSelectionDomain.maximumCoinsPerRound, len(keptResults))]
}
