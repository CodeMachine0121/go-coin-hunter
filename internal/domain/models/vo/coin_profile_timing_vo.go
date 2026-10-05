package vo

import "time"

// CoinProfileTimingVo bounds how long gathering profiles may take: each source call has its own timeout, and the
// whole gathering must end within the base budget plus one allowance per security lookup.
type CoinProfileTimingVo struct {
	SourceRequestTimeout    time.Duration
	RoundBaseBudget         time.Duration
	SecurityLookupAllowance time.Duration
}
