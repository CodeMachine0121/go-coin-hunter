package vo

// CoinInsightAnswerVo is the analyst's answer as given, before any of it is trusted.
type CoinInsightAnswerVo struct {
	Direction string
	Strength  int
	Catalyst  string
	Risks     []string
	Evidence  []string
	DataGaps  []string
}
