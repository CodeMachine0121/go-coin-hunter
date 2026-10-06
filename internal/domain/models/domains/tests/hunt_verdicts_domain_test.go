package domains_test

import (
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func verdictPolicy() vo.HuntVerdictPolicyVo {
	return vo.HuntVerdictPolicyVo{MinimumLeverage: 1, MaximumLeverage: 5, MaximumPositionSizePercent: decimal.NewFromInt(10),
		MinimumStopLossPercent: decimal.NewFromInt(1), MaximumStopLossPercent: decimal.NewFromInt(50),
		MinimumTakeProfitPercent: decimal.NewFromInt(1), MaximumTakeProfitPercent: decimal.NewFromInt(200),
		MinimumBullishInsightStrength: 6, MinimumHuntBoardConfidence: 50}
}

func pricedMaterial(coinSymbol string, lastPrice string) vo.HuntVerdictMaterialVo {
	price := decimal.RequireFromString(lastPrice)
	return vo.HuntVerdictMaterialVo{CoinSymbol: coinSymbol, MarketStructure: &vo.PerpetualMarketStructureVo{ExchangeName: "幣安", LastPrice: &price}}
}

func answerFor(coinSymbol, action string, confidence, leverage int, positionSize, stopLoss, takeProfit string) vo.HuntVerdictAnswerVo {
	return vo.HuntVerdictAnswerVo{CoinSymbol: coinSymbol, Action: action, Confidence: confidence, Leverage: leverage,
		PositionSizePercent: decimal.RequireFromString(positionSize), StopLossPercent: decimal.RequireFromString(stopLoss),
		TakeProfitPercent: decimal.RequireFromString(takeProfit), Rationale: " 理由 ", ConflictResolution: " 無明顯矛盾 "}
}

func text(value *decimal.Decimal) string {
	if value == nil {
		return "none"
	}
	return value.String()
}

func onlyVerdict(t *testing.T, material vo.HuntVerdictMaterialVo, answer vo.HuntVerdictAnswerVo) entities.CoinVerdict {
	coinVerdicts := domains.NewHuntVerdictsDomain([]vo.HuntVerdictMaterialVo{material}, []vo.HuntVerdictAnswerVo{answer}, verdictPolicy()).ToCoinVerdicts(3)
	require.Len(t, coinVerdicts, 1)
	return coinVerdicts[0]
}

func TestHuntVerdictClampsTheNumbers(t *testing.T) {
	coinVerdict := onlyVerdict(t, pricedMaterial("PENGU", "0.01"), answerFor("PENGU", "long", 130, 20, "30", "10", "30"))

	assert.Equal(t, "long", coinVerdict.Action)
	assert.Equal(t, 100, coinVerdict.Confidence)
	assert.Equal(t, 5, coinVerdict.Leverage)
	assert.Equal(t, "0.1", coinVerdict.PositionSizeRatio.String())
	assert.Equal(t, "理由", coinVerdict.Rationale)
	assert.Equal(t, "無明顯矛盾", coinVerdict.ConflictResolution)
	assert.Equal(t, uint(3), coinVerdict.PipelineRunID)

	low := onlyVerdict(t, pricedMaterial("PENGU", "0.01"), answerFor("PENGU", "long", -5, 0, "-2", "60", "500"))
	assert.Equal(t, 0, low.Confidence)
	assert.Equal(t, 1, low.Leverage)
	assert.Equal(t, "0", low.PositionSizeRatio.String())
	assert.Equal(t, "0.5", text(low.StopLossRatio))
	assert.Equal(t, "2", text(low.TakeProfitRatio))
}

func TestHuntVerdictWidensATakeProfitUnderOnePercent(t *testing.T) {
	coinVerdict := onlyVerdict(t, pricedMaterial("PENGU", "0.01"), answerFor("PENGU", "long", 70, 3, "5", "10", "0.2"))

	assert.Equal(t, "0.01", text(coinVerdict.TakeProfitRatio))
	assert.Equal(t, "0.0101", text(coinVerdict.TakeProfitPrice))
}

func TestHuntVerdictPricesFollowTheDirection(t *testing.T) {
	testCases := []struct {
		name                                   string
		action, stopLoss                       string
		wantStopLossPrice, wantTakeProfitPrice string
		wantStopLossRatio                      string
	}{
		{name: "a long stops below and takes profit above", action: "long", stopLoss: "10", wantStopLossPrice: "0.009", wantTakeProfitPrice: "0.013", wantStopLossRatio: "0.1"},
		{name: "a stop under one percent is widened to one percent", action: "long", stopLoss: "0.5", wantStopLossPrice: "0.0099", wantTakeProfitPrice: "0.013", wantStopLossRatio: "0.01"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			coinVerdict := onlyVerdict(t, pricedMaterial("PENGU", "0.01"), answerFor("PENGU", testCase.action, 70, 3, "5", testCase.stopLoss, "30"))

			assert.Equal(t, testCase.wantStopLossPrice, text(coinVerdict.StopLossPrice))
			assert.Equal(t, testCase.wantTakeProfitPrice, text(coinVerdict.TakeProfitPrice))
			assert.Equal(t, testCase.wantStopLossRatio, text(coinVerdict.StopLossRatio))
			assert.Equal(t, "0.01", text(coinVerdict.ReferencePrice))
			assert.Equal(t, "0.05", coinVerdict.PositionSizeRatio.String())
		})
	}
}

func TestHuntVerdictWatchesWhatCannotBeActedOn(t *testing.T) {
	testCases := []struct {
		name          string
		material      vo.HuntVerdictMaterialVo
		answer        vo.HuntVerdictAnswerVo
		wantAction    string
		wantRationale string
	}{
		{name: "an unknown action", material: pricedMaterial("PENGU", "0.01"), answer: answerFor("PENGU", "梭哈", 80, 5, "10", "10", "30"), wantAction: "watch", wantRationale: "理由"},
		{name: "avoid is kept without numbers", material: pricedMaterial("PENGU", "0.01"), answer: answerFor("PENGU", " AVOID ", 60, 5, "10", "10", "30"), wantAction: "avoid", wantRationale: "理由"},
		{name: "watch is kept without numbers", material: pricedMaterial("PENGU", "0.01"), answer: answerFor("PENGU", "watch", 60, 5, "10", "10", "30"), wantAction: "watch", wantRationale: "理由"},
		{name: "a long without a price", material: vo.HuntVerdictMaterialVo{CoinSymbol: "PENGU"}, answer: answerFor("PENGU", "long", 80, 5, "10", "10", "30"),
			wantAction: "watch", wantRationale: "查不到最新價格，無法設定停損"},
		{name: "a long at a price of zero", material: pricedMaterial("PENGU", "0"), answer: answerFor("PENGU", "long", 80, 5, "10", "10", "30"),
			wantAction: "watch", wantRationale: "查不到最新價格，無法設定停損"},
		{name: "a long with a market but no price", material: vo.HuntVerdictMaterialVo{CoinSymbol: "PENGU", MarketStructure: &vo.PerpetualMarketStructureVo{ExchangeName: "OKX"}},
			answer: answerFor("PENGU", "long", 80, 5, "10", "10", "30"), wantAction: "watch", wantRationale: "查不到最新價格，無法設定停損"},
		{name: "a short, since the hunt only goes long", material: pricedMaterial("PENGU", "0.01"), answer: answerFor("PENGU", "short", 80, 3, "5", "10", "30"),
			wantAction: "watch", wantRationale: "理由"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			coinVerdict := onlyVerdict(t, testCase.material, testCase.answer)

			assert.Equal(t, testCase.wantAction, coinVerdict.Action)
			assert.Equal(t, testCase.wantRationale, coinVerdict.Rationale)
			assert.Equal(t, 0, coinVerdict.Leverage)
			assert.True(t, coinVerdict.PositionSizeRatio.IsZero())
			assert.Nil(t, coinVerdict.StopLossPrice)
			assert.Nil(t, coinVerdict.TakeProfitPrice)
			assert.Nil(t, coinVerdict.ReferencePrice)
		})
	}
}

func TestHuntVerdictsCoverExactlyTheCoinsShown(t *testing.T) {
	coinVerdicts := domains.NewHuntVerdictsDomain(
		[]vo.HuntVerdictMaterialVo{pricedMaterial("PENGU", "0.01"), pricedMaterial("STRK", "0.2")},
		[]vo.HuntVerdictAnswerVo{answerFor("doge", "long", 90, 5, "10", "10", "30"), answerFor(" pengu ", "long", 70, 3, "5", "10", "30"),
			answerFor("PENGU", "short", 10, 1, "1", "10", "30")},
		verdictPolicy()).ToCoinVerdicts(3)

	require.Len(t, coinVerdicts, 2)
	assert.Equal(t, "PENGU", coinVerdicts[0].CoinSymbol)
	assert.Equal(t, "long", coinVerdicts[0].Action)
	assert.Equal(t, entities.CoinVerdict{PipelineRunID: 3, CoinSymbol: "STRK", Action: "watch", PositionSizeRatio: decimal.Zero, Rationale: "CIO 未給出裁決"}, coinVerdicts[1])
}

func TestConcludeVerdict(t *testing.T) {
	succeeded := domains.NewPipelineRunDomain(entities.PipelineRun{ID: 1, Status: "running"}).ConcludeVerdict(2, receivedAt)
	assert.Equal(t, string(vo.PipelineRunStatusSucceeded), succeeded.Status)
	assert.Equal(t, receivedAt, *succeeded.FinishedAt)

	noData := domains.NewPipelineRunDomain(entities.PipelineRun{ID: 1, Status: "running"}).ConcludeVerdict(0, receivedAt)
	assert.Equal(t, string(vo.PipelineRunStatusNoData), noData.Status)
	assert.Equal(t, receivedAt, *noData.FinishedAt)
}

func insight(coinSymbol string, succeeded bool, direction string, strength int) entities.CoinInsight {
	return entities.CoinInsight{CoinSymbol: coinSymbol, Succeeded: succeeded, Direction: direction, Strength: strength}
}

func TestBullishFocusHandsOnlyStrongBullishInsightsToTheStrategist(t *testing.T) {
	bullishInsights := domains.NewBullishFocusDomain(verdictPolicy()).SelectBullishInsights([]entities.CoinInsight{
		insight("PENGU", true, "bullish", 8),
		insight("STRK", true, "bearish", 9),
		insight("ARB", true, "bullish", 6),
		insight("OP", true, "bullish", 5),
		insight("TIA", true, "neutral", 9),
		insight("SEI", false, "bullish", 9),
	})

	symbols := []string{}
	for _, bullishInsight := range bullishInsights {
		symbols = append(symbols, bullishInsight.CoinSymbol)
	}
	assert.Equal(t, []string{"PENGU", "ARB"}, symbols)
	assert.Empty(t, domains.NewBullishFocusDomain(verdictPolicy()).SelectBullishInsights([]entities.CoinInsight{insight("TIA", true, "neutral", 9)}))
}

func TestBullishFocusPutsOnlyConfidentLongsOnTheHuntBoard(t *testing.T) {
	calculatedAt := receivedAt
	huntBoardEntries := domains.NewBullishFocusDomain(verdictPolicy()).ToHuntBoardEntries([]entities.CoinVerdict{
		{PipelineRunID: 3, CoinSymbol: "PENGU", Action: "long", Confidence: 72, Leverage: 3, Rationale: "理由"},
		{PipelineRunID: 3, CoinSymbol: "STRK", Action: "watch", Confidence: 90},
		{PipelineRunID: 3, CoinSymbol: "ARB", Action: "avoid", Confidence: 90},
		{PipelineRunID: 3, CoinSymbol: "OP", Action: "long", Confidence: 50},
		{PipelineRunID: 3, CoinSymbol: "TIA", Action: "long", Confidence: 49},
	}, calculatedAt)

	require.Len(t, huntBoardEntries, 2)
	assert.Equal(t, entities.HuntBoardEntry{CoinSymbol: "PENGU", CalculatedAt: calculatedAt, PipelineRunID: 3, Action: "long", Confidence: 72,
		Leverage: 3, Rationale: "理由"}, huntBoardEntries[0])
	assert.Equal(t, "OP", huntBoardEntries[1].CoinSymbol)
	assert.Empty(t, domains.NewBullishFocusDomain(verdictPolicy()).ToHuntBoardEntries(
		[]entities.CoinVerdict{{CoinSymbol: "STRK", Action: "watch"}}, calculatedAt))
}

func TestBullishFocusKeepsItsThresholdsOnTheirScales(t *testing.T) {
	offScale := verdictPolicy()
	offScale.MinimumBullishInsightStrength, offScale.MinimumHuntBoardConfidence = 11, 150
	strictest := domains.NewBullishFocusDomain(offScale)

	assert.Len(t, strictest.SelectBullishInsights([]entities.CoinInsight{insight("PENGU", true, "bullish", 10)}), 1, "a strength over 10 is read as 10")
	assert.Len(t, strictest.ToHuntBoardEntries([]entities.CoinVerdict{{CoinSymbol: "PENGU", Action: "long", Confidence: 100}}, receivedAt), 1,
		"a confidence over 100 is read as 100")

	offScale.MinimumBullishInsightStrength, offScale.MinimumHuntBoardConfidence = -3, -20
	loosest := domains.NewBullishFocusDomain(offScale)
	assert.Empty(t, loosest.SelectBullishInsights([]entities.CoinInsight{insight("PENGU", true, "bullish", 0)}), "a strength under 1 is read as 1")
	assert.Len(t, loosest.ToHuntBoardEntries([]entities.CoinVerdict{{CoinSymbol: "PENGU", Action: "long", Confidence: 0}}, receivedAt), 1)
}
