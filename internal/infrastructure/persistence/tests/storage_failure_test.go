package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newClosedDatabase is a database whose connection is already gone, so every storage call reports failure.
func newClosedDatabase(t *testing.T) *gorm.DB {
	database := newMigratedDatabase(t)
	connection, connectionError := database.DB()
	require.NoError(t, connectionError)
	require.NoError(t, connection.Close())

	return database
}

func TestRepositoriesReportStorageFailures(t *testing.T) {
	database := newClosedDatabase(t)
	executionContext := context.Background()
	pipelineRunRepository := persistence.NewPipelineRunRepository(database)
	coinIntelligenceRepository := persistence.NewCoinIntelligenceRepository(database)
	coinCandidateRepository := persistence.NewCoinCandidateRepository(database)
	outcomeRepository := persistence.NewInformationSourceOutcomeRepository(database)

	failures := map[string]error{}
	_, failures["create run"] = pipelineRunRepository.Create(executionContext, entities.PipelineRun{StartedAt: time.Now()})
	failures["update run"] = pipelineRunRepository.Update(executionContext, entities.PipelineRun{ID: 1})
	_, failures["find run"] = pipelineRunRepository.FindOne(executionContext, 1)
	_, _, failures["find latest run"] = pipelineRunRepository.FindLatestSucceeded(executionContext, "discovery")
	_, failures["find runs"] = pipelineRunRepository.FindAll(executionContext)
	_, failures["find running runs"] = pipelineRunRepository.FindRunning(executionContext)
	failures["save intelligences"] = coinIntelligenceRepository.SaveNew(executionContext, []entities.CoinIntelligence{{SourceName: "s"}})
	_, failures["find window"] = coinIntelligenceRepository.FindPublishedSince(executionContext, time.Now())
	_, failures["find run intelligences"] = coinIntelligenceRepository.FindByPipelineRunID(executionContext, 1)
	failures["create candidates"] = coinCandidateRepository.CreateAll(executionContext, []entities.CoinCandidate{{CoinSymbol: "CT"}})
	_, failures["find candidates"] = coinCandidateRepository.FindByPipelineRunID(executionContext, 1)
	failures["create outcomes"] = outcomeRepository.CreateAll(executionContext, []entities.InformationSourceOutcome{{SourceName: "s"}})
	_, failures["find declared contracts"] = coinIntelligenceRepository.FindDeclaredContractAddresses(executionContext, []string{"CT"})
	failures["create filter results"] = persistence.NewCoinFilterResultRepository(database).CreateAll(executionContext, []entities.CoinFilterResult{{CoinSymbol: "CT"}})
	_, failures["find filter results"] = persistence.NewCoinFilterResultRepository(database).FindByPipelineRunID(executionContext, 1)
	_, failures["find coin intelligence"] = coinIntelligenceRepository.FindByCoinSymbolsSince(executionContext, []string{"CT"}, time.Now())
	failures["create insights"] = persistence.NewCoinInsightRepository(database).CreateAll(executionContext, []entities.CoinInsight{{CoinSymbol: "CT"}})
	_, failures["find insights"] = persistence.NewCoinInsightRepository(database).FindByPipelineRunID(executionContext, 1)
	failures["migrate"] = persistence.NewSchemaMigrator(database).Migrate()

	for step, failure := range failures {
		assert.Error(t, failure, step)
	}
}

func TestEmptyBatchesNeverTouchStorage(t *testing.T) {
	database := newClosedDatabase(t)

	assert.NoError(t, persistence.NewCoinIntelligenceRepository(database).SaveNew(context.Background(), nil))
	assert.NoError(t, persistence.NewInformationSourceOutcomeRepository(database).CreateAll(context.Background(), nil))
	assert.NoError(t, persistence.NewCoinCandidateRepository(database).CreateAll(context.Background(), nil))
	assert.NoError(t, persistence.NewCoinFilterResultRepository(database).CreateAll(context.Background(), nil))
	assert.NoError(t, persistence.NewCoinInsightRepository(database).CreateAll(context.Background(), nil))
}

func TestNewDatabaseFailsWhenTheServerIsUnreachable(t *testing.T) {
	_, openError := persistence.NewDatabase("host=127.0.0.1 port=1 user=postgres password=postgres dbname=none_test sslmode=disable connect_timeout=2")

	assert.Error(t, openError)
}
