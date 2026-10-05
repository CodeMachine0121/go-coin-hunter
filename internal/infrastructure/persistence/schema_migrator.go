package persistence

import (
	"fmt"

	"gorm.io/gorm"
)

// SchemaMigrator syncs the schema from the entities; the loose element type is required by GORM's AutoMigrate.
type SchemaMigrator struct {
	database *gorm.DB
}

func NewSchemaMigrator(database *gorm.DB) *SchemaMigrator {
	return &SchemaMigrator{database: database}
}

func (schemaMigrator *SchemaMigrator) Migrate() error {
	if migrateError := schemaMigrator.database.AutoMigrate(); migrateError != nil {
		return fmt.Errorf("auto migrate schema: %w", migrateError)
	}

	return nil
}
