package persistence

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// slowQueryThreshold matches GORM's default; the custom logger exists only for the not-found flag.
const slowQueryThreshold = 200 * time.Millisecond

// NewDatabase opens the SQLite file, creating its folder first, and keeps "record not found" out of the log because empty reads are expected answers here.
func NewDatabase(databasePath string) (*gorm.DB, error) {
	if mkdirError := os.MkdirAll(filepath.Dir(databasePath), 0o755); mkdirError != nil {
		return nil, fmt.Errorf("create sqlite folder: %w", mkdirError)
	}

	database, openError := gorm.Open(sqlite.Open(databasePath), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             slowQueryThreshold,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if openError != nil {
		return nil, fmt.Errorf("open sqlite database: %w", openError)
	}

	return database, nil
}
