package domains_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestSelectCandidatesAggregatesPerCoin(t *testing.T) {
	startedAt := receivedAt
	selection := domains.NewCoinCandidateSelectionDomain(vo.DiscoveryPolicyVo{Window: 72 * time.Hour, ExcludedCoinSymbols: []string{"eth"}})

	coinCandidates := selection.SelectCandidates(9, startedAt, []entities.CoinIntelligence{
		{SourceName: "bybitAnnouncement", CoinSymbol: "CT", PublishedAt: startedAt.Add(-2 * time.Hour)},
		{SourceName: "binanceAnnouncement", CoinSymbol: "CT", PublishedAt: startedAt.Add(-5 * time.Hour)},
		{SourceName: "binanceAnnouncement", CoinSymbol: "CT", PublishedAt: startedAt.Add(-1 * time.Hour)},
		{SourceName: "coinGeckoTrending", CoinSymbol: "ETH", PublishedAt: startedAt},
		{SourceName: "binanceAnnouncement", CoinSymbol: "", PublishedAt: startedAt},
	})

	assert.Equal(t, []entities.CoinCandidate{{
		PipelineRunID: 9, CoinSymbol: "CT", SourceCount: 2, IntelligenceCount: 3, EarliestMentionedAt: startedAt.Add(-5 * time.Hour),
	}}, coinCandidates)
	assert.Equal(t, startedAt.Add(-72*time.Hour), selection.WindowStart(startedAt))
}

func TestSelectCandidatesTreatsOneSymbolOnTwoChainsAsOneCoin(t *testing.T) {
	selection := domains.NewCoinCandidateSelectionDomain(vo.DiscoveryPolicyVo{Window: 72 * time.Hour})

	coinCandidates := selection.SelectCandidates(1, receivedAt, []entities.CoinIntelligence{
		{SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "solana:tok1", CoinSymbol: "PEPE2", PublishedAt: receivedAt},
		{SourceName: "dexScreenerTokenProfile", ExternalIdentifier: "ethereum:0xtok2", CoinSymbol: "PEPE2", PublishedAt: receivedAt},
	})

	assert.Len(t, coinCandidates, 1)
	assert.Equal(t, 1, coinCandidates[0].SourceCount)
	assert.Equal(t, 2, coinCandidates[0].IntelligenceCount)
}
