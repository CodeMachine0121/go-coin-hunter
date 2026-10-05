package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoinInsightRepositoryKeepsListsWithTheInsight(t *testing.T) {
	coinInsightRepository := persistence.NewCoinInsightRepository(newMigratedDatabase(t))
	require.NoError(t, coinInsightRepository.CreateAll(context.Background(), []entities.CoinInsight{
		{PipelineRunID: 4, CoinSymbol: "STRK", Succeeded: true, Direction: "neutral", Strength: 4, Risks: []string{"解鎖"}, Evidence: []string{"持倉 -3%"}, DataGaps: []string{"查不到近期新聞"}, MarketStructureExchange: "Bybit"},
		{PipelineRunID: 4, CoinSymbol: "PENGU", FailureReason: "AI 回覆格式不合格", Risks: []string{}, Evidence: []string{}, DataGaps: []string{}},
		{PipelineRunID: 5, CoinSymbol: "ZORA", Succeeded: true, Risks: []string{}, Evidence: []string{}, DataGaps: []string{}},
	}))
	require.NoError(t, coinInsightRepository.CreateAll(context.Background(), nil))

	coinInsights, findError := coinInsightRepository.FindByPipelineRunID(context.Background(), 4)

	require.NoError(t, findError)
	require.Len(t, coinInsights, 2)
	assert.Equal(t, "PENGU", coinInsights[0].CoinSymbol)
	assert.Equal(t, "AI 回覆格式不合格", coinInsights[0].FailureReason)
	assert.Equal(t, []string{"解鎖"}, coinInsights[1].Risks)
	assert.Equal(t, []string{"查不到近期新聞"}, coinInsights[1].DataGaps)
	assert.Error(t, coinInsightRepository.CreateAll(context.Background(), []entities.CoinInsight{
		{PipelineRunID: 4, CoinSymbol: "STRK", Risks: []string{}, Evidence: []string{}, DataGaps: []string{}}}))
}

func TestCoinIntelligenceRepositoryFindsRecentIntelligenceOfCoins(t *testing.T) {
	coinIntelligenceRepository := persistence.NewCoinIntelligenceRepository(newMigratedDatabase(t))
	require.NoError(t, coinIntelligenceRepository.SaveNew(context.Background(), []entities.CoinIntelligence{
		{PipelineRunID: 1, SourceName: "s", ExternalIdentifier: "old", CoinSymbol: "PENGU", PublishedAt: storedAt.Add(-time.Hour)},
		{PipelineRunID: 1, SourceName: "s", ExternalIdentifier: "new", CoinSymbol: "PENGU", PublishedAt: storedAt},
		{PipelineRunID: 1, SourceName: "s", ExternalIdentifier: "stale", CoinSymbol: "PENGU", PublishedAt: storedAt.Add(-100 * time.Hour)},
		{PipelineRunID: 1, SourceName: "s", ExternalIdentifier: "other", CoinSymbol: "ZORA", PublishedAt: storedAt},
	}))

	coinIntelligences, findError := coinIntelligenceRepository.FindByCoinSymbolsSince(context.Background(), []string{"PENGU"}, storedAt.Add(-72*time.Hour))

	require.NoError(t, findError)
	identifiers := []string{}
	for _, coinIntelligence := range coinIntelligences {
		identifiers = append(identifiers, coinIntelligence.ExternalIdentifier)
	}
	assert.Equal(t, []string{"new", "old"}, identifiers)
}
