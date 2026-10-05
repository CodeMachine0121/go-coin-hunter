package dto

type CoinFilterResultDto struct {
	PipelineRunID uint                   `json:"pipelineRunId"`
	CoinSymbol    string                 `json:"coinSymbol"`
	IsKept        bool                   `json:"isKept"`
	Verdicts      []CoinFilterVerdictDto `json:"verdicts"`
}

type CoinFilterVerdictDto struct {
	FilterName string `json:"filterName"`
	Outcome    string `json:"outcome"`
	Reason     string `json:"reason"`
}
