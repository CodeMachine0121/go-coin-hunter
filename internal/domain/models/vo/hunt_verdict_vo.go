package vo

import "github.com/shopspring/decimal"

type HuntActionVo string

const (
	HuntActionLong  HuntActionVo = "long"
	HuntActionShort HuntActionVo = "short"
	HuntActionWatch HuntActionVo = "watch"
	HuntActionAvoid HuntActionVo = "avoid"
)

// HuntVerdictMaterialVo is one coin as the strategist sees it: its insight and the market as it is now.
type HuntVerdictMaterialVo struct {
	CoinSymbol      string
	Direction       string
	Strength        int
	Catalyst        string
	Risks           []string
	Evidence        []string
	DataGaps        []string
	MarketStructure *PerpetualMarketStructureVo
}

// HuntVerdictAnswerVo is the strategist's answer for one coin, before any of it is trusted; percentages are as given (10 means 10%).
type HuntVerdictAnswerVo struct {
	CoinSymbol          string
	Action              string
	Confidence          int
	Leverage            int
	PositionSizePercent decimal.Decimal
	StopLossPercent     decimal.Decimal
	TakeProfitPercent   decimal.Decimal
	Rationale           string
	ConflictResolution  string
}

// HuntVerdictPolicyVo holds the safe ranges every verdict is clamped into; percentages as whole numbers (10 means 10%).
type HuntVerdictPolicyVo struct {
	MinimumLeverage            int
	MaximumLeverage            int
	MaximumPositionSizePercent decimal.Decimal
	MinimumStopLossPercent     decimal.Decimal
	MaximumStopLossPercent     decimal.Decimal
	MinimumTakeProfitPercent   decimal.Decimal
	MaximumTakeProfitPercent   decimal.Decimal
}
