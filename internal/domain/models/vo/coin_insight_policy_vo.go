package vo

import "time"

// CoinInsightPolicyVo bounds one insight round: how much is gathered, how many coins and how many at once.
type CoinInsightPolicyVo struct {
	MaximumCoinsPerRound        int
	MaximumConcurrentAnalyses   int
	IntelligenceWindow          time.Duration
	MaximumIntelligenceHeadline int
	NewsLookback                time.Duration
	MaximumNewsHeadlines        int
	SourceRequestTimeout        time.Duration
}
