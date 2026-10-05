package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boardEntry(coinSymbol string, confidence int, calculatedAt time.Time, pipelineRunID uint) entities.HuntBoardEntry {
	stopLossPrice := decimal.RequireFromString("0.009")
	return entities.HuntBoardEntry{CoinSymbol: coinSymbol, CalculatedAt: calculatedAt, PipelineRunID: pipelineRunID, Action: "long",
		Confidence: confidence, Leverage: 3, PositionSizeRatio: decimal.RequireFromString("0.05"), StopLossPrice: &stopLossPrice, Rationale: "第一輪"}
}

func boardBySymbol(t *testing.T, huntBoardRepository *persistence.HuntBoardRepository) map[string]entities.HuntBoardEntry {
	huntBoardEntries, findError := huntBoardRepository.FindAll(context.Background())
	require.NoError(t, findError)
	bySymbol := map[string]entities.HuntBoardEntry{}
	for _, huntBoardEntry := range huntBoardEntries {
		bySymbol[huntBoardEntry.CoinSymbol] = huntBoardEntry
	}
	return bySymbol
}

func TestHuntBoardRewriteOverwritesThisRoundAndRemovesTheRest(t *testing.T) {
	huntBoardRepository := persistence.NewHuntBoardRepository(newMigratedDatabase(t))
	firstRound := storedAt
	secondRound := storedAt.Add(4 * time.Hour)
	require.NoError(t, huntBoardRepository.Rewrite(context.Background(), []entities.HuntBoardEntry{
		boardEntry("BTC", 50, firstRound, 1), boardEntry("ETH", 60, firstRound, 1), boardEntry("BNB", 70, firstRound, 1),
	}))
	firstBoard := boardBySymbol(t, huntBoardRepository)
	require.Len(t, firstBoard, 3)

	updatedBtc := boardEntry("BTC", 90, secondRound, 2)
	updatedBtc.Action, updatedBtc.Rationale, updatedBtc.StopLossPrice = "short", "第二輪", nil
	require.NoError(t, huntBoardRepository.Rewrite(context.Background(), []entities.HuntBoardEntry{updatedBtc, boardEntry("ETH", 40, secondRound, 2)}))

	secondBoard := boardBySymbol(t, huntBoardRepository)
	assert.Len(t, secondBoard, 2)
	assert.NotContains(t, secondBoard, "BNB")
	btc := secondBoard["BTC"]
	assert.Equal(t, firstBoard["BTC"].ID, btc.ID)
	assert.True(t, secondRound.Equal(btc.CalculatedAt))
	assert.Equal(t, uint(2), btc.PipelineRunID)
	assert.Equal(t, "short", btc.Action)
	assert.Equal(t, 90, btc.Confidence)
	assert.Equal(t, "第二輪", btc.Rationale)
	assert.Nil(t, btc.StopLossPrice)
	assert.True(t, secondRound.Equal(secondBoard["ETH"].CalculatedAt))
	assert.Equal(t, 40, secondBoard["ETH"].Confidence)
}

func TestHuntBoardListsHighestConfidenceFirst(t *testing.T) {
	huntBoardRepository := persistence.NewHuntBoardRepository(newMigratedDatabase(t))
	require.NoError(t, huntBoardRepository.Rewrite(context.Background(), []entities.HuntBoardEntry{
		boardEntry("STRK", 40, storedAt, 1), boardEntry("PENGU", 80, storedAt, 1), boardEntry("AAA", 80, storedAt, 1),
	}))

	huntBoardEntries, findError := huntBoardRepository.FindAll(context.Background())

	require.NoError(t, findError)
	symbols := []string{}
	for _, huntBoardEntry := range huntBoardEntries {
		symbols = append(symbols, huntBoardEntry.CoinSymbol)
	}
	assert.Equal(t, []string{"AAA", "PENGU", "STRK"}, symbols)
	assert.Equal(t, "0.009", huntBoardEntries[0].StopLossPrice.String())
}

func TestHuntBoardRewriteWithNothingEmptiesTheBoard(t *testing.T) {
	huntBoardRepository := persistence.NewHuntBoardRepository(newMigratedDatabase(t))
	require.NoError(t, huntBoardRepository.Rewrite(context.Background(), []entities.HuntBoardEntry{boardEntry("BTC", 50, storedAt, 1)}))

	require.NoError(t, huntBoardRepository.Rewrite(context.Background(), nil))

	assert.Empty(t, boardBySymbol(t, huntBoardRepository))
}

func TestHuntBoardRewriteFailingLeavesTheBoardAsItWas(t *testing.T) {
	huntBoardRepository := persistence.NewHuntBoardRepository(newMigratedDatabase(t))
	require.NoError(t, huntBoardRepository.Rewrite(context.Background(), []entities.HuntBoardEntry{boardEntry("BTC", 50, storedAt, 1), boardEntry("BNB", 60, storedAt, 1)}))

	// The same coin twice in one round cannot be written, so the whole rewrite is rolled back.
	rewriteError := huntBoardRepository.Rewrite(context.Background(), []entities.HuntBoardEntry{boardEntry("ETH", 50, storedAt, 2), boardEntry("ETH", 60, storedAt, 2)})

	assert.Error(t, rewriteError)
	board := boardBySymbol(t, huntBoardRepository)
	assert.Len(t, board, 2)
	assert.Contains(t, board, "BNB")
	assert.Equal(t, uint(1), board["BTC"].PipelineRunID)
}

func TestCoinVerdictRepositoryKeepsEachRunsVerdicts(t *testing.T) {
	coinVerdictRepository := persistence.NewCoinVerdictRepository(newMigratedDatabase(t))
	stopLossPrice := decimal.RequireFromString("0.0099")
	require.NoError(t, coinVerdictRepository.CreateAll(context.Background(), []entities.CoinVerdict{
		{PipelineRunID: 3, CoinSymbol: "STRK", Action: "watch", PositionSizeRatio: decimal.Zero, Rationale: "CIO 未給出裁決"},
		{PipelineRunID: 3, CoinSymbol: "PENGU", Action: "long", Confidence: 70, Leverage: 3, PositionSizeRatio: decimal.RequireFromString("0.05"), StopLossPrice: &stopLossPrice},
		{PipelineRunID: 4, CoinSymbol: "ZORA", Action: "avoid", PositionSizeRatio: decimal.Zero},
	}))
	require.NoError(t, coinVerdictRepository.CreateAll(context.Background(), nil))

	coinVerdicts, findError := coinVerdictRepository.FindByPipelineRunID(context.Background(), 3)

	require.NoError(t, findError)
	require.Len(t, coinVerdicts, 2)
	assert.Equal(t, "PENGU", coinVerdicts[0].CoinSymbol)
	assert.Equal(t, "0.0099", coinVerdicts[0].StopLossPrice.String())
	assert.Equal(t, "0.05", coinVerdicts[0].PositionSizeRatio.String())
	assert.Equal(t, "CIO 未給出裁決", coinVerdicts[1].Rationale)
	assert.Nil(t, coinVerdicts[1].StopLossPrice)
}

func TestHuntBoardRewriteReportsAFailedRemoval(t *testing.T) {
	database := newMigratedDatabase(t)
	require.NoError(t, database.Migrator().DropTable(&entities.HuntBoardEntry{}))

	rewriteError := persistence.NewHuntBoardRepository(database).Rewrite(context.Background(), []entities.HuntBoardEntry{boardEntry("BTC", 50, storedAt, 1)})

	assert.ErrorContains(t, rewriteError, "remove coins absent from this round")
}
