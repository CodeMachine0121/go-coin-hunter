package persistence

import (
	"context"
	"fmt"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HuntBoardRepository struct {
	database *gorm.DB
}

func NewHuntBoardRepository(database *gorm.DB) *HuntBoardRepository {
	return &HuntBoardRepository{database: database}
}

// Rewrite upserts each entry by coin symbol, keeping its row id, then removes every coin not among the entries.
func (huntBoardRepository *HuntBoardRepository) Rewrite(executionContext context.Context, huntBoardEntries []entities.HuntBoardEntry) error {
	rewriteError := huntBoardRepository.database.WithContext(executionContext).Transaction(func(transaction *gorm.DB) error {
		// GORM's NOT IN clause takes the loose element type.
		keptCoinSymbols := make([]any, 0, len(huntBoardEntries))
		for _, huntBoardEntry := range huntBoardEntries {
			keptCoinSymbols = append(keptCoinSymbols, huntBoardEntry.CoinSymbol)
		}
		removal := transaction.Session(&gorm.Session{AllowGlobalUpdate: true})
		if len(keptCoinSymbols) > 0 {
			removal = removal.Where(clause.Not(clause.IN{Column: "coin_symbol", Values: keptCoinSymbols}))
		}
		if deleteError := removal.Delete(&entities.HuntBoardEntry{}).Error; deleteError != nil {
			return fmt.Errorf("remove coins absent from this round: %w", deleteError)
		}
		if len(huntBoardEntries) == 0 {
			return nil
		}

		return transaction.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "coin_symbol"}},
			DoUpdates: clause.AssignmentColumns([]string{"calculated_at", "pipeline_run_id", "action", "confidence", "leverage",
				"position_size_ratio", "reference_price", "stop_loss_ratio", "stop_loss_price", "take_profit_ratio",
				"take_profit_price", "rationale", "conflict_resolution"}),
		}).Create(&huntBoardEntries).Error
	})
	if rewriteError != nil {
		return fmt.Errorf("rewrite hunt board: %w", rewriteError)
	}

	return nil
}

func (huntBoardRepository *HuntBoardRepository) FindAll(executionContext context.Context) ([]entities.HuntBoardEntry, error) {
	huntBoardEntries := []entities.HuntBoardEntry{}
	if findError := huntBoardRepository.database.WithContext(executionContext).
		Order("confidence DESC").Order("coin_symbol").Find(&huntBoardEntries).Error; findError != nil {
		return nil, fmt.Errorf("find hunt board: %w", findError)
	}

	return huntBoardEntries, nil
}
