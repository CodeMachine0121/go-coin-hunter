package domains_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

var receivedAt = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

func TestInformationItemCoinIntelligences(t *testing.T) {
	publishedAt := receivedAt.Add(-time.Hour)
	testCases := []struct {
		name            string
		informationItem vo.InformationItemVo
		wantCoinSymbols []string
		wantPublishedAt time.Time
	}{
		{
			name:            "declared symbols win over the title and are upper-cased",
			informationItem: vo.InformationItemVo{Title: "Listing (XYZ)", DeclaredCoinSymbols: []string{" pump "}, PublishedAt: &publishedAt},
			wantCoinSymbols: []string{"PUMP"}, wantPublishedAt: publishedAt,
		},
		{
			name:            "an announcement naming two coins is one intelligence per coin",
			informationItem: vo.InformationItemVo{Title: "Binance Will List Zora (ZORA) and Pump (PUMP)", PublishedAt: &publishedAt},
			wantCoinSymbols: []string{"ZORA", "PUMP"}, wantPublishedAt: publishedAt,
		},
		{
			name:            "no publish time counts as first received",
			informationItem: vo.InformationItemVo{DeclaredCoinSymbols: []string{"FET"}},
			wantCoinSymbols: []string{"FET"}, wantPublishedAt: receivedAt,
		},
		{
			name:            "no coin keeps one intelligence without a coin",
			informationItem: vo.InformationItemVo{Title: "系統維護通知", DeclaredCoinSymbols: []string{"  "}, PublishedAt: &publishedAt},
			wantCoinSymbols: []string{""}, wantPublishedAt: publishedAt,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			coinIntelligences := domains.NewInformationItemDomain(testCase.informationItem).CoinIntelligences(5, receivedAt)

			coinSymbols := []string{}
			for _, coinIntelligence := range coinIntelligences {
				coinSymbols = append(coinSymbols, coinIntelligence.CoinSymbol)
				assert.Equal(t, uint(5), coinIntelligence.PipelineRunID)
				assert.Equal(t, testCase.wantPublishedAt, coinIntelligence.PublishedAt)
			}
			assert.Equal(t, testCase.wantCoinSymbols, coinSymbols)
		})
	}
}
