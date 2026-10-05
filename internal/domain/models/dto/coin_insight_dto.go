package dto

type CoinInsightDto struct {
	PipelineRunID           uint     `json:"pipelineRunId"`
	CoinSymbol              string   `json:"coinSymbol"`
	Succeeded               bool     `json:"succeeded"`
	FailureReason           string   `json:"failureReason"`
	Direction               string   `json:"direction"`
	Strength                int      `json:"strength"`
	Catalyst                string   `json:"catalyst"`
	Risks                   []string `json:"risks"`
	Evidence                []string `json:"evidence"`
	DataGaps                []string `json:"dataGaps"`
	MarketStructureExchange string   `json:"marketStructureExchange"`
}
