package vo

import "time"

// CoinProfileVo is everything the filtering rules may read about one candidate, gathered before any rule runs.
type CoinProfileVo struct {
	CoinSymbol  string
	EvaluatedAt time.Time
	MarketData  *CoinMarketDataVo
	// ContractAddress is the contract the security check looked at, or nil when the coin has none on a supported chain.
	ContractAddress *TokenAddressVo
	TokenSecurity   *TokenSecurityVo
	// TokenSecurityNotQueriedReason explains a missing TokenSecurity that was skipped on purpose rather than unknown.
	TokenSecurityNotQueriedReason string
	PerpetualContractExchanges    []string
	UnlockScheduleKnown           bool
	UnlockEvents                  []TokenUnlockEventVo
	// MarketStructure is the coin's perpetual as it trades now, or nil when no exchange answers for it.
	MarketStructure *PerpetualMarketStructureVo
}
