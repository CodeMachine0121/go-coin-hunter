package persistence_test

import (
	"path/filepath"
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/persistence"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newMigratedDatabase opens a throwaway SQLite file per test so storage behavior is checked against the real engine.
func newMigratedDatabase(t *testing.T) *gorm.DB {
	database, openError := persistence.NewDatabase(filepath.Join(t.TempDir(), "test.sqlite3"))
	require.NoError(t, openError)
	require.NoError(t, persistence.NewSchemaMigrator(database).Migrate())

	return database
}
