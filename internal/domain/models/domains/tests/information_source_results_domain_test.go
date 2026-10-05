package domains_test

import (
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestInformationSourceResultsReadIntoOutcomesAndIntelligence(t *testing.T) {
	publishedAt := receivedAt
	results := domains.NewInformationSourceResultsDomain([]vo.InformationSourceResultVo{
		{SourceName: "binanceAnnouncement", InformationItems: []vo.InformationItemVo{
			{SourceName: "binanceAnnouncement", ExternalIdentifier: "a1", Title: "Binance Will List Cotton (CT)", PublishedAt: &publishedAt},
		}},
		{SourceName: "okxAnnouncement", FailureReason: "連線逾時"},
	})

	assert.True(t, results.AnySourceSucceeded())
	assert.Equal(t, []entities.InformationSourceOutcome{
		{PipelineRunID: 3, SourceName: "binanceAnnouncement", Succeeded: true, InformationItemCount: 1},
		{PipelineRunID: 3, SourceName: "okxAnnouncement", Succeeded: false, FailureReason: "連線逾時"},
	}, results.InformationSourceOutcomes(3))
	coinIntelligences := results.CoinIntelligences(3, receivedAt)
	assert.Len(t, coinIntelligences, 1)
	assert.Equal(t, "CT", coinIntelligences[0].CoinSymbol)
}

func TestInformationSourceResultsWithEverySourceFailing(t *testing.T) {
	results := domains.NewInformationSourceResultsDomain([]vo.InformationSourceResultVo{{SourceName: "a", FailureReason: "x"}})

	assert.False(t, results.AnySourceSucceeded())
	assert.Empty(t, results.CoinIntelligences(1, receivedAt))
}
