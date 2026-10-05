package persistence_test

import (
	"context"
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoinFilterResultRepositoryKeepsVerdictsWithTheirResult(t *testing.T) {
	coinFilterResultRepository := persistence.NewCoinFilterResultRepository(newMigratedDatabase(t))
	require.NoError(t, coinFilterResultRepository.CreateAll(context.Background(), []entities.CoinFilterResult{
		{PipelineRunID: 3, CoinSymbol: "PUMP", IsKept: false, Verdicts: []entities.CoinFilterVerdictRecord{
			{FilterName: "fullyDilutedValuation", Outcome: "rejected", Reason: "完全稀釋估值 52 億美元高於上限 10 億美元"}}},
		{PipelineRunID: 3, CoinSymbol: "GRASS", IsKept: true, Verdicts: []entities.CoinFilterVerdictRecord{{FilterName: "unlockSchedule", Outcome: "noData", Reason: "查不到解鎖時程"}}},
		{PipelineRunID: 4, CoinSymbol: "ZORA", IsKept: true, Verdicts: []entities.CoinFilterVerdictRecord{}},
	}))
	require.NoError(t, coinFilterResultRepository.CreateAll(context.Background(), nil))

	coinFilterResults, findError := coinFilterResultRepository.FindByPipelineRunID(context.Background(), 3)

	require.NoError(t, findError)
	require.Len(t, coinFilterResults, 2)
	assert.Equal(t, "GRASS", coinFilterResults[0].CoinSymbol)
	assert.True(t, coinFilterResults[0].IsKept)
	assert.Equal(t, []entities.CoinFilterVerdictRecord{{FilterName: "fullyDilutedValuation", Outcome: "rejected",
		Reason: "完全稀釋估值 52 億美元高於上限 10 億美元"}}, coinFilterResults[1].Verdicts)
}

func TestCoinIntelligenceRepositoryFindsTheLatestDeclaredContract(t *testing.T) {
	coinIntelligenceRepository := persistence.NewCoinIntelligenceRepository(newMigratedDatabase(t))
	require.NoError(t, coinIntelligenceRepository.SaveNew(context.Background(), []entities.CoinIntelligence{
		{PipelineRunID: 1, SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "solana:Old", CoinSymbol: "DOUU", PublishedAt: storedAt.Add(-1), ChainID: "solana", ContractAddress: "Old"},
		{PipelineRunID: 1, SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "solana:NewMint", CoinSymbol: "DOUU", PublishedAt: storedAt, ChainID: "solana", ContractAddress: "NewMint"},
		{PipelineRunID: 1, SourceName: "binanceAnnouncement", ExternalIdentifier: "a1", CoinSymbol: "ZORA", PublishedAt: storedAt},
		{PipelineRunID: 1, SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "base:0xother", CoinSymbol: "OTHER", PublishedAt: storedAt, ChainID: "base", ContractAddress: "0xother"},
	}))

	declaredContractAddresses, findError := coinIntelligenceRepository.FindDeclaredContractAddresses(context.Background(), []string{"DOUU", "ZORA"})

	require.NoError(t, findError)
	assert.Equal(t, map[string]vo.TokenAddressVo{"DOUU": {ChainID: "solana", Address: "NewMint"}}, declaredContractAddresses)
}
