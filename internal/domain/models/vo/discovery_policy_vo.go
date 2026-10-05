package vo

import "time"

// DiscoveryPolicyVo holds the operator-tunable rules of one discovery round.
type DiscoveryPolicyVo struct {
	Window               time.Duration
	ExcludedCoinSymbols  []string
	ItemLimitPerSource   int
	SourceRequestTimeout time.Duration
}
