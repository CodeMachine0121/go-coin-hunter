package dto

import "time"

type CoinCandidateDto struct {
	PipelineRunID       uint      `json:"pipelineRunId"`
	CoinSymbol          string    `json:"coinSymbol"`
	SourceCount         int       `json:"sourceCount"`
	IntelligenceCount   int       `json:"intelligenceCount"`
	EarliestMentionedAt time.Time `json:"earliestMentionedAt"`
}
