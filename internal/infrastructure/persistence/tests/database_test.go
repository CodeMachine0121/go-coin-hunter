package persistence_test

import (
	"os"
	"strings"
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// testDatabaseNameSuffix is required on the test database name because these tests empty every table.
const testDatabaseNameSuffix = "_test"

// newMigratedDatabase connects via TEST_POSTGRES_DSN, migrates and empties the tables, skipping when the variable is unset.
func newMigratedDatabase(t *testing.T) *gorm.DB {
	dataSourceName := os.Getenv("TEST_POSTGRES_DSN")
	if dataSourceName == "" {
		t.Skip("TEST_POSTGRES_DSN is not set; skipping storage tests")
	}
	databaseName := ""
	for _, setting := range strings.Fields(dataSourceName) {
		if name, found := strings.CutPrefix(setting, "dbname="); found {
			databaseName = name
		}
	}
	require.True(t, strings.HasSuffix(databaseName, testDatabaseNameSuffix),
		"TEST_POSTGRES_DSN 指向的資料庫名稱必須以 %s 結尾——這些測試每次都會清空資料表", testDatabaseNameSuffix)

	database, openError := persistence.NewDatabase(dataSourceName)
	require.NoError(t, openError)
	// Close each pool so a long test run does not exhaust the server's connections.
	t.Cleanup(func() {
		if connection, connectionError := database.DB(); connectionError == nil {
			_ = connection.Close()
		}
	})
	require.NoError(t, persistence.NewSchemaMigrator(database).Migrate())

	clearedDatabase := database.Session(&gorm.Session{AllowGlobalUpdate: true})
	for _, entity := range []any{&entities.CoinInsight{}, &entities.CoinFilterResult{}, &entities.CoinCandidate{}, &entities.CoinIntelligence{}, &entities.InformationSourceOutcome{}, &entities.PipelineRun{}} {
		require.NoError(t, clearedDatabase.Delete(entity).Error)
	}

	return database
}
