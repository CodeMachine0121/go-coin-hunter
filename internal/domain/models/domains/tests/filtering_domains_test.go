package domains_test

import (
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNumberDescriptions(t *testing.T) {
	testCases := []struct {
		value, wantUsd, wantQuantity, wantPercentage string
	}{
		{value: "5200000000", wantUsd: "52 億美元", wantQuantity: "52 億", wantPercentage: "520000000000%"},
		{value: "10000000", wantUsd: "1,000 萬美元", wantQuantity: "1,000 萬", wantPercentage: "1000000000%"},
		{value: "990000", wantUsd: "99 萬美元", wantQuantity: "99 萬", wantPercentage: "99000000%"},
		{value: "123456789", wantUsd: "1.23 億美元", wantQuantity: "1.23 億", wantPercentage: "12345678900%"},
		{value: "9999", wantUsd: "9,999 美元", wantQuantity: "9,999", wantPercentage: "999900%"},
		{value: "-25000", wantUsd: "-2.5 萬美元", wantQuantity: "-2.5 萬", wantPercentage: "-2500000%"},
		{value: "0.105", wantUsd: "0.11 美元", wantQuantity: "0.11", wantPercentage: "10.5%"},
		{value: "0.00102", wantUsd: "0 美元", wantQuantity: "0", wantPercentage: "0.102%"},
		{value: "0.0000123456", wantUsd: "0 美元", wantQuantity: "0", wantPercentage: "0.0012%"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.value, func(t *testing.T) {
			description := domains.NewNumberDescriptionDomain(decimal.RequireFromString(testCase.value))

			assert.Equal(t, testCase.wantUsd, description.AsUsd())
			assert.Equal(t, testCase.wantQuantity, description.AsTokenQuantity())
			assert.Equal(t, testCase.wantPercentage, description.AsPercentage())
		})
	}
}

func TestCoinFilterVerdictsKeepUnlessRejected(t *testing.T) {
	passed := vo.FilterVerdictVo{FilterName: "a", Outcome: vo.FilterOutcomePassed}
	noData := vo.FilterVerdictVo{FilterName: "b", Outcome: vo.FilterOutcomeNoData, Reason: "查不到解鎖時程"}
	rejected := vo.FilterVerdictVo{FilterName: "c", Outcome: vo.FilterOutcomeRejected, Reason: "x"}

	assert.True(t, domains.NewCoinFilterVerdictsDomain([]vo.FilterVerdictVo{passed, noData}).IsKept())
	assert.False(t, domains.NewCoinFilterVerdictsDomain([]vo.FilterVerdictVo{passed, rejected}).IsKept())
	assert.Equal(t, entities.CoinFilterResult{PipelineRunID: 4, CoinSymbol: "GRASS", IsKept: true, Verdicts: []entities.CoinFilterVerdictRecord{
		{FilterName: "a", Outcome: "passed"}, {FilterName: "b", Outcome: "noData", Reason: "查不到解鎖時程"},
	}}, domains.NewCoinFilterVerdictsDomain([]vo.FilterVerdictVo{passed, noData}).ToCoinFilterResult(4, "GRASS"))
}

func TestTokenAddressComparisonKey(t *testing.T) {
	assert.Equal(t, "ethereum:0xabcdef", domains.NewTokenAddressDomain(vo.TokenAddressVo{ChainID: vo.ChainEthereum, Address: "0xAbCdEf"}).ComparisonKey())
	assert.Equal(t, "solana:TokA", domains.NewTokenAddressDomain(vo.TokenAddressVo{ChainID: vo.ChainSolana, Address: "TokA"}).ComparisonKey())
}

func TestConcludeFiltering(t *testing.T) {
	running := domains.NewPipelineRunDomain(entities.PipelineRun{ID: 1, Status: string(vo.PipelineRunStatusRunning)})

	assert.Equal(t, string(vo.PipelineRunStatusSucceeded), running.ConcludeFiltering(1, receivedAt).Status)
	assert.Equal(t, string(vo.PipelineRunStatusNoData), running.ConcludeFiltering(0, receivedAt).Status)
	assert.Equal(t, receivedAt, *running.ConcludeFiltering(0, receivedAt).FinishedAt)
}
