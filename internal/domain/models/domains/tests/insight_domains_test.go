package domains_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/stretchr/testify/assert"
)

func TestCoinInsightAnswerNormalization(t *testing.T) {
	material := vo.CoinInsightMaterialVo{CoinSymbol: "PENGU", DataGaps: []string{"查不到近期新聞"},
		MarketStructure: &vo.PerpetualMarketStructureVo{ExchangeName: "幣安"}}
	testCases := []struct {
		name          string
		direction     string
		strength      int
		wantDirection string
		wantStrength  int
	}{
		{name: "known values are kept", direction: "bullish", strength: 7, wantDirection: "bullish", wantStrength: 7},
		{name: "bearish is kept", direction: " Bearish ", strength: 3, wantDirection: "bearish", wantStrength: 3},
		{name: "strength above ten is clamped", direction: "bullish", strength: 12, wantDirection: "bullish", wantStrength: 10},
		{name: "strength below one is clamped", direction: "bullish", strength: 0, wantDirection: "bullish", wantStrength: 1},
		{name: "an unknown direction is neutral", direction: "暴漲", strength: 5, wantDirection: "neutral", wantStrength: 5},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			coinInsight := domains.NewCoinInsightAnswerDomain(vo.CoinInsightAnswerVo{Direction: testCase.direction, Strength: testCase.strength}, material).ToCoinInsight(3)

			assert.Equal(t, testCase.wantDirection, coinInsight.Direction)
			assert.Equal(t, testCase.wantStrength, coinInsight.Strength)
		})
	}
}

func TestCoinInsightAnswerKeepsTheRestOfTheAnswer(t *testing.T) {
	material := vo.CoinInsightMaterialVo{CoinSymbol: "PENGU", DataGaps: []string{"查不到近期新聞"},
		MarketStructure: &vo.PerpetualMarketStructureVo{ExchangeName: "幣安"}}

	coinInsight := domains.NewCoinInsightAnswerDomain(vo.CoinInsightAnswerVo{
		Direction: "bullish", Strength: 7, Catalyst: "  幣安上新永續合約 ", Risks: []string{"解鎖"}, Evidence: []string{"持倉 +12%"},
		DataGaps: []string{"查不到近期新聞", " ", "缺鏈上持幣分佈"},
	}, material).ToCoinInsight(3)

	assert.Equal(t, entities.CoinInsight{PipelineRunID: 3, CoinSymbol: "PENGU", Succeeded: true, Direction: "bullish", Strength: 7,
		Catalyst: "幣安上新永續合約", Risks: []string{"解鎖"}, Evidence: []string{"持倉 +12%"},
		DataGaps: []string{"查不到近期新聞", "缺鏈上持幣分佈"}, MarketStructureExchange: "幣安"}, coinInsight)
	assert.Equal(t, "", domains.NewCoinInsightAnswerDomain(vo.CoinInsightAnswerVo{}, vo.CoinInsightMaterialVo{}).ToCoinInsight(1).MarketStructureExchange)
}

func TestInsightCandidateSelection(t *testing.T) {
	base := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	coinFilterResults := []entities.CoinFilterResult{
		{CoinSymbol: "LATE", IsKept: true}, {CoinSymbol: "DROPPED", IsKept: false},
		{CoinSymbol: "BETA", IsKept: true}, {CoinSymbol: "ALPHA", IsKept: true}, {CoinSymbol: "EARLY", IsKept: true},
	}
	coinCandidates := []entities.CoinCandidate{
		{CoinSymbol: "LATE", EarliestMentionedAt: base.Add(3 * time.Hour)}, {CoinSymbol: "DROPPED", EarliestMentionedAt: base},
		{CoinSymbol: "BETA", EarliestMentionedAt: base.Add(time.Hour)}, {CoinSymbol: "ALPHA", EarliestMentionedAt: base.Add(time.Hour)},
		{CoinSymbol: "EARLY", EarliestMentionedAt: base},
	}

	selected := domains.NewInsightCandidateSelectionDomain(3).SelectKeptResults(coinFilterResults, coinCandidates)

	symbols := []string{}
	for _, coinFilterResult := range selected {
		symbols = append(symbols, coinFilterResult.CoinSymbol)
	}
	assert.Equal(t, []string{"EARLY", "ALPHA", "BETA"}, symbols)
	assert.Len(t, domains.NewInsightCandidateSelectionDomain(20).SelectKeptResults(coinFilterResults, coinCandidates), 4)
}

func TestConcludeInsight(t *testing.T) {
	running := domains.NewPipelineRunDomain(entities.PipelineRun{ID: 1, Status: string(vo.PipelineRunStatusRunning)})

	assert.Equal(t, string(vo.PipelineRunStatusSucceeded), running.ConcludeInsight(1, receivedAt).Status)
	failed := running.ConcludeInsight(0, receivedAt)
	assert.Equal(t, string(vo.PipelineRunStatusFailed), failed.Status)
	assert.Equal(t, "所有候選幣分析失敗", failed.FailureReason)
}
