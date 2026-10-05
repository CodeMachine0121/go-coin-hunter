package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storedAt = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

func discoveryRun(status vo.PipelineRunStatusVo, startedAt time.Time) entities.PipelineRun {
	return entities.PipelineRun{Step: string(vo.PipelineRunStepDiscovery), TriggerSource: string(vo.PipelineRunTriggerSourceManual),
		Status: string(status), StartedAt: startedAt}
}

func TestCoinIntelligenceRepositorySavesEachMessageOnce(t *testing.T) {
	coinIntelligenceRepository := persistence.NewCoinIntelligenceRepository(newMigratedDatabase(t))
	coinIntelligence := entities.CoinIntelligence{PipelineRunID: 1, SourceName: "binanceAnnouncement", ExternalIdentifier: "a1", CoinSymbol: "CT", PublishedAt: storedAt}

	require.NoError(t, coinIntelligenceRepository.SaveNew(context.Background(), []entities.CoinIntelligence{coinIntelligence}))
	coinIntelligence.PipelineRunID = 2
	require.NoError(t, coinIntelligenceRepository.SaveNew(context.Background(), []entities.CoinIntelligence{coinIntelligence}))

	saved, findError := coinIntelligenceRepository.FindPublishedSince(context.Background(), storedAt.Add(-time.Hour))
	require.NoError(t, findError)
	require.Len(t, saved, 1)
	assert.Equal(t, uint(1), saved[0].PipelineRunID)
}

func TestCoinIntelligenceRepositoryFindsTheWindowInclusively(t *testing.T) {
	coinIntelligenceRepository := persistence.NewCoinIntelligenceRepository(newMigratedDatabase(t))
	windowStart := storedAt.Add(-72 * time.Hour)
	require.NoError(t, coinIntelligenceRepository.SaveNew(context.Background(), []entities.CoinIntelligence{
		{PipelineRunID: 1, SourceName: "s", ExternalIdentifier: "at-boundary", CoinSymbol: "CT", PublishedAt: windowStart},
		{PipelineRunID: 1, SourceName: "s", ExternalIdentifier: "just-outside", CoinSymbol: "CT", PublishedAt: windowStart.Add(-time.Minute)},
		{PipelineRunID: 2, SourceName: "s", ExternalIdentifier: "inside", CoinSymbol: "PUMP", PublishedAt: storedAt},
	}))

	inWindow, findError := coinIntelligenceRepository.FindPublishedSince(context.Background(), windowStart)
	require.NoError(t, findError)
	identifiers := []string{}
	for _, coinIntelligence := range inWindow {
		identifiers = append(identifiers, coinIntelligence.ExternalIdentifier)
	}
	assert.Equal(t, []string{"at-boundary", "inside"}, identifiers)

	ofRunTwo, runError := coinIntelligenceRepository.FindByPipelineRunID(context.Background(), 2)
	require.NoError(t, runError)
	require.Len(t, ofRunTwo, 1)
	assert.Equal(t, "PUMP", ofRunTwo[0].CoinSymbol)
}

func TestPipelineRunRepositoryFindsTheLatestSucceededRun(t *testing.T) {
	testCases := []struct {
		name      string
		runs      []entities.PipelineRun
		wantFound bool
		wantIndex int
	}{
		{
			name:      "the newer of two succeeded runs",
			runs:      []entities.PipelineRun{discoveryRun(vo.PipelineRunStatusSucceeded, storedAt.Add(-time.Hour)), discoveryRun(vo.PipelineRunStatusSucceeded, storedAt)},
			wantFound: true, wantIndex: 1,
		},
		{
			name:      "a later failed run does not replace it",
			runs:      []entities.PipelineRun{discoveryRun(vo.PipelineRunStatusSucceeded, storedAt.Add(-time.Hour)), discoveryRun(vo.PipelineRunStatusFailed, storedAt)},
			wantFound: true, wantIndex: 0,
		},
		{name: "never succeeded", runs: []entities.PipelineRun{discoveryRun(vo.PipelineRunStatusNoData, storedAt)}, wantFound: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pipelineRunRepository := persistence.NewPipelineRunRepository(newMigratedDatabase(t))
			createdIDs := []uint{}
			for _, run := range testCase.runs {
				created, createError := pipelineRunRepository.Create(context.Background(), run)
				require.NoError(t, createError)
				createdIDs = append(createdIDs, created.ID)
			}

			latest, found, findError := pipelineRunRepository.FindLatestSucceeded(context.Background(), string(vo.PipelineRunStepDiscovery))

			require.NoError(t, findError)
			assert.Equal(t, testCase.wantFound, found)
			if testCase.wantFound {
				assert.Equal(t, createdIDs[testCase.wantIndex], latest.ID)
			}
		})
	}
}

func TestPipelineRunRepositoryListsNewestFirstWithOutcomes(t *testing.T) {
	database := newMigratedDatabase(t)
	pipelineRunRepository := persistence.NewPipelineRunRepository(database)
	older, _ := pipelineRunRepository.Create(context.Background(), discoveryRun(vo.PipelineRunStatusSucceeded, storedAt.Add(-time.Hour)))
	newer, _ := pipelineRunRepository.Create(context.Background(), discoveryRun(vo.PipelineRunStatusRunning, storedAt))
	require.NoError(t, persistence.NewInformationSourceOutcomeRepository(database).CreateAll(context.Background(), []entities.InformationSourceOutcome{
		{PipelineRunID: older.ID, SourceName: "okxAnnouncement", FailureReason: "連線逾時"},
	}))

	pipelineRuns, findError := pipelineRunRepository.FindAll(context.Background())
	require.NoError(t, findError)
	require.Len(t, pipelineRuns, 2)
	assert.Equal(t, newer.ID, pipelineRuns[0].ID)
	assert.Equal(t, older.ID, pipelineRuns[1].ID)
	require.Len(t, pipelineRuns[1].InformationSourceOutcomes, 1)
	assert.Equal(t, "連線逾時", pipelineRuns[1].InformationSourceOutcomes[0].FailureReason)

	running, runningError := pipelineRunRepository.FindRunning(context.Background())
	require.NoError(t, runningError)
	require.Len(t, running, 1)
	assert.Equal(t, newer.ID, running[0].ID)

	newer.Status = string(vo.PipelineRunStatusFailed)
	require.NoError(t, pipelineRunRepository.Update(context.Background(), newer))
	reloaded, findOneError := pipelineRunRepository.FindOne(context.Background(), newer.ID)
	require.NoError(t, findOneError)
	assert.Equal(t, string(vo.PipelineRunStatusFailed), reloaded.Status)

	_, missingError := pipelineRunRepository.FindOne(context.Background(), 999)
	assert.ErrorIs(t, missingError, domains.ErrPipelineRunNotFound)
}

func TestCoinCandidateRepositoryListsARunsCandidatesBySymbol(t *testing.T) {
	coinCandidateRepository := persistence.NewCoinCandidateRepository(newMigratedDatabase(t))
	require.NoError(t, coinCandidateRepository.CreateAll(context.Background(), []entities.CoinCandidate{
		{PipelineRunID: 1, CoinSymbol: "PUMP", EarliestMentionedAt: storedAt},
		{PipelineRunID: 1, CoinSymbol: "CT", EarliestMentionedAt: storedAt},
		{PipelineRunID: 2, CoinSymbol: "ZORA", EarliestMentionedAt: storedAt},
	}))
	require.NoError(t, coinCandidateRepository.CreateAll(context.Background(), nil))

	coinCandidates, findError := coinCandidateRepository.FindByPipelineRunID(context.Background(), 1)

	require.NoError(t, findError)
	require.Len(t, coinCandidates, 2)
	assert.Equal(t, "CT", coinCandidates[0].CoinSymbol)
	assert.Equal(t, "PUMP", coinCandidates[1].CoinSymbol)
}

func TestCoinIntelligenceRepositoryTellsMessagesApartBySourceMessageAndCoin(t *testing.T) {
	coinIntelligenceRepository := persistence.NewCoinIntelligenceRepository(newMigratedDatabase(t))
	original := entities.CoinIntelligence{PipelineRunID: 1, SourceName: "binanceAnnouncement", ExternalIdentifier: "a1", CoinSymbol: "ZORA", PublishedAt: storedAt}
	sameMessageOtherCoin := original
	sameMessageOtherCoin.CoinSymbol = "PUMP"
	sameIdentifierOtherSource := original
	sameIdentifierOtherSource.SourceName = "bybitAnnouncement"

	require.NoError(t, coinIntelligenceRepository.SaveNew(context.Background(),
		[]entities.CoinIntelligence{original, sameMessageOtherCoin, sameIdentifierOtherSource}))

	saved, findError := coinIntelligenceRepository.FindByPipelineRunID(context.Background(), 1)
	require.NoError(t, findError)
	assert.Len(t, saved, 3)
}
