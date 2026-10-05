package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CoinIntelligenceRepository struct {
	database *gorm.DB
}

func NewCoinIntelligenceRepository(database *gorm.DB) *CoinIntelligenceRepository {
	return &CoinIntelligenceRepository{database: database}
}

func (coinIntelligenceRepository *CoinIntelligenceRepository) SaveNew(
	executionContext context.Context, coinIntelligences []entities.CoinIntelligence,
) error {
	if len(coinIntelligences) == 0 {
		return nil
	}
	if saveError := coinIntelligenceRepository.database.WithContext(executionContext).
		Clauses(clause.OnConflict{DoNothing: true}).Create(&coinIntelligences).Error; saveError != nil {
		return fmt.Errorf("save coin intelligences: %w", saveError)
	}

	return nil
}

func (coinIntelligenceRepository *CoinIntelligenceRepository) FindPublishedSince(
	executionContext context.Context, publishedSince time.Time,
) ([]entities.CoinIntelligence, error) {
	coinIntelligences := []entities.CoinIntelligence{}
	if findError := coinIntelligenceRepository.database.WithContext(executionContext).
		Where(clause.Gte{Column: "published_at", Value: publishedSince.UTC()}).Order("published_at").Order("id").
		Find(&coinIntelligences).Error; findError != nil {
		return nil, fmt.Errorf("find coin intelligences published since: %w", findError)
	}

	return coinIntelligences, nil
}

func (coinIntelligenceRepository *CoinIntelligenceRepository) FindByPipelineRunID(
	executionContext context.Context, pipelineRunID uint,
) ([]entities.CoinIntelligence, error) {
	coinIntelligences := []entities.CoinIntelligence{}
	if findError := coinIntelligenceRepository.database.WithContext(executionContext).
		Where(&entities.CoinIntelligence{PipelineRunID: pipelineRunID}).Order("published_at DESC").Order("id").
		Find(&coinIntelligences).Error; findError != nil {
		return nil, fmt.Errorf("find coin intelligences of pipeline run: %w", findError)
	}

	return coinIntelligences, nil
}

func (coinIntelligenceRepository *CoinIntelligenceRepository) FindDeclaredContractAddresses(
	executionContext context.Context, coinSymbols []string,
) (map[string]vo.TokenAddressVo, error) {
	// GORM's IN clause takes the loose element type.
	coinSymbolValues := make([]any, 0, len(coinSymbols))
	for _, coinSymbol := range coinSymbols {
		coinSymbolValues = append(coinSymbolValues, coinSymbol)
	}
	coinIntelligences := []entities.CoinIntelligence{}
	if findError := coinIntelligenceRepository.database.WithContext(executionContext).
		Where(clause.IN{Column: "coin_symbol", Values: coinSymbolValues}).
		Where(clause.Neq{Column: "contract_address", Value: ""}).
		Order("published_at").Order("id").
		Find(&coinIntelligences).Error; findError != nil {
		return nil, fmt.Errorf("find declared contract addresses: %w", findError)
	}

	// Ordered oldest first, so the most recently declared address is the one left in the map.
	declaredContractAddresses := map[string]vo.TokenAddressVo{}
	for _, coinIntelligence := range coinIntelligences {
		declaredContractAddresses[coinIntelligence.CoinSymbol] = vo.TokenAddressVo{
			ChainID: coinIntelligence.ChainID, Address: coinIntelligence.ContractAddress,
		}
	}

	return declaredContractAddresses, nil
}

func (coinIntelligenceRepository *CoinIntelligenceRepository) FindByCoinSymbolsSince(
	executionContext context.Context, coinSymbols []string, publishedSince time.Time,
) ([]entities.CoinIntelligence, error) {
	// GORM's IN clause takes the loose element type.
	coinSymbolValues := make([]any, 0, len(coinSymbols))
	for _, coinSymbol := range coinSymbols {
		coinSymbolValues = append(coinSymbolValues, coinSymbol)
	}
	coinIntelligences := []entities.CoinIntelligence{}
	if findError := coinIntelligenceRepository.database.WithContext(executionContext).
		Where(clause.IN{Column: "coin_symbol", Values: coinSymbolValues}).
		Where(clause.Gte{Column: "published_at", Value: publishedSince.UTC()}).
		Order("published_at DESC").Order("id").
		Find(&coinIntelligences).Error; findError != nil {
		return nil, fmt.Errorf("find coin intelligences of coins: %w", findError)
	}

	return coinIntelligences, nil
}
