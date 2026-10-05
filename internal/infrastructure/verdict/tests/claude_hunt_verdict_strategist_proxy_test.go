package verdict_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/verdict"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMessagesApi struct {
	lastBody     map[string]any
	lastHeader   http.Header
	requestCount int
}

func (fakeApi *fakeMessagesApi) serve(t *testing.T, statusCode int, stopReason string, answerText string) string {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fakeApi.requestCount++
		body, _ := io.ReadAll(request.Body)
		fakeApi.lastBody = map[string]any{}
		_ = json.Unmarshal(body, &fakeApi.lastBody)
		fakeApi.lastHeader = request.Header.Clone()
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(statusCode)
		if statusCode != http.StatusOK {
			_, _ = writer.Write([]byte(`{"type":"error","error":{"type":"overloaded_error","message":"overloaded"}}`))
			return
		}
		textJson, _ := json.Marshal(answerText)
		_, _ = writer.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","content":[{"type":"text","text":` +
			string(textJson) + `}],"stop_reason":"` + stopReason + `","usage":{"input_tokens":10,"output_tokens":10}}`))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

const readableVerdicts = `{"verdicts":[{"coinSymbol":"PENGU","action":"long","confidence":72,"leverage":3,"positionSizePercent":4.5,
	"stopLossPercent":8,"takeProfitPercent":25.5,"rationale":"幣安上新合約","conflictResolution":"無明顯矛盾"}]}`

func pengu() []vo.HuntVerdictMaterialVo {
	lastPrice := decimal.RequireFromString("0.0097")
	return []vo.HuntVerdictMaterialVo{{CoinSymbol: "PENGU", Direction: "bullish", Strength: 7, Catalyst: "上新合約", Risks: []string{"解鎖"},
		Evidence: []string{"持倉 +12%"}, DataGaps: []string{}, MarketStructure: &vo.PerpetualMarketStructureVo{ExchangeName: "幣安", LastPrice: &lastPrice}}}
}

func TestClaudeStrategistReadsVerdictsAndAsksInTheAgreedShape(t *testing.T) {
	fakeApi := &fakeMessagesApi{}
	proxy := verdict.NewClaudeHuntVerdictStrategistProxy("test-key", fakeApi.serve(t, http.StatusOK, "end_turn", readableVerdicts), "claude-opus-5-5", "high", 5*time.Second)

	answers, synthesizeError := proxy.SynthesizeVerdicts(context.Background(), pengu())

	require.NoError(t, synthesizeError)
	require.Len(t, answers, 1)
	assert.Equal(t, "PENGU", answers[0].CoinSymbol)
	assert.Equal(t, "long", answers[0].Action)
	assert.Equal(t, 72, answers[0].Confidence)
	assert.Equal(t, 3, answers[0].Leverage)
	assert.Equal(t, "4.5", answers[0].PositionSizePercent.String())
	assert.Equal(t, "8", answers[0].StopLossPercent.String())
	assert.Equal(t, "25.5", answers[0].TakeProfitPercent.String())
	assert.Equal(t, "幣安上新合約", answers[0].Rationale)
	assert.Equal(t, "無明顯矛盾", answers[0].ConflictResolution)

	assert.Equal(t, 1, fakeApi.requestCount)
	assert.Equal(t, "claude-opus-5-5", fakeApi.lastBody["model"])
	assert.Equal(t, "default", fakeApi.lastBody["fallbacks"])
	assert.Contains(t, fakeApi.lastHeader.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01")
	outputConfig := fakeApi.lastBody["output_config"].(map[string]any)
	assert.Equal(t, "high", outputConfig["effort"])
	assert.Equal(t, "json_schema", outputConfig["format"].(map[string]any)["type"])
	assert.Equal(t, "ephemeral", fakeApi.lastBody["system"].([]any)[0].(map[string]any)["cache_control"].(map[string]any)["type"])
	userText := fakeApi.lastBody["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	materials := []map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(userText), &materials))
	assert.Equal(t, "PENGU", materials[0]["coinSymbol"])
	assert.Equal(t, "0.0097", materials[0]["marketStructure"].(map[string]any)["lastPrice"])
	assert.Nil(t, materials[0]["marketStructure"].(map[string]any)["fundingRate"])
}

func TestClaudeStrategistTellsUnusableAnswersFromServiceErrors(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		stopReason   string
		answerText   string
		wantUnusable bool
	}{
		{name: "a refusal, even one carrying readable verdicts", statusCode: http.StatusOK, stopReason: "refusal", answerText: readableVerdicts, wantUnusable: true},
		{name: "a cut-off answer", statusCode: http.StatusOK, stopReason: "max_tokens", answerText: readableVerdicts, wantUnusable: true},
		{name: "text that is not json", statusCode: http.StatusOK, stopReason: "end_turn", answerText: "做多", wantUnusable: true},
		{name: "no verdict list", statusCode: http.StatusOK, stopReason: "end_turn", answerText: `{}`, wantUnusable: true},
		{name: "a verdict missing a field", statusCode: http.StatusOK, stopReason: "end_turn", answerText: `{"verdicts":[{"coinSymbol":"PENGU","action":"long"}]}`, wantUnusable: true},
		{name: "a verdict missing a number", statusCode: http.StatusOK, stopReason: "end_turn",
			answerText: `{"verdicts":[{"coinSymbol":"PENGU","action":"long","confidence":1,"leverage":1,"positionSizePercent":1,"stopLossPercent":1,"rationale":"","conflictResolution":""}]}`, wantUnusable: true},
		{name: "a verdict with a non-numeric number", statusCode: http.StatusOK, stopReason: "end_turn",
			answerText: `{"verdicts":[{"coinSymbol":"PENGU","action":"long","confidence":"high","leverage":1,"positionSizePercent":1,"stopLossPercent":1,"takeProfitPercent":1,"rationale":"","conflictResolution":""}]}`, wantUnusable: true},
		{name: "an overloaded service", statusCode: http.StatusServiceUnavailable, wantUnusable: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeApi := &fakeMessagesApi{}
			proxy := verdict.NewClaudeHuntVerdictStrategistProxy("sk-ant-secret", fakeApi.serve(t, testCase.statusCode, testCase.stopReason, testCase.answerText),
				"claude-opus-5-5", "high", 5*time.Second)

			_, synthesizeError := proxy.SynthesizeVerdicts(context.Background(), pengu())

			require.Error(t, synthesizeError)
			assert.Equal(t, testCase.wantUnusable, errors.Is(synthesizeError, domains.ErrHuntVerdictAnswerUnusable))
			assert.Equal(t, 1, fakeApi.requestCount)
			assert.NotContains(t, synthesizeError.Error(), "sk-ant-secret")
		})
	}
}

func TestClaudeStrategistGivesUpAtItsDeadline(t *testing.T) {
	released := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-released }))
	t.Cleanup(func() {
		close(released)
		server.Close()
	})

	_, synthesizeError := verdict.NewClaudeHuntVerdictStrategistProxy("test-key", server.URL, "claude-opus-5-5", "high", 100*time.Millisecond).
		SynthesizeVerdicts(context.Background(), pengu())

	assert.ErrorIs(t, synthesizeError, context.DeadlineExceeded)
}

func TestClaudeStrategistUsesTheDefaultEndpointWithoutABaseAddress(t *testing.T) {
	assert.NotNil(t, verdict.NewClaudeHuntVerdictStrategistProxy("test-key", "", "claude-opus-5-5", "high", time.Second))
}

func TestClaudeStrategistShowsACoinWithoutMarketStructure(t *testing.T) {
	fakeApi := &fakeMessagesApi{}
	proxy := verdict.NewClaudeHuntVerdictStrategistProxy("test-key", fakeApi.serve(t, http.StatusOK, "end_turn", `{"verdicts":[]}`), "claude-opus-5-5", "high", 5*time.Second)

	answers, synthesizeError := proxy.SynthesizeVerdicts(context.Background(), []vo.HuntVerdictMaterialVo{{CoinSymbol: "PONS"}})

	require.NoError(t, synthesizeError)
	assert.Empty(t, answers)
	userText := fakeApi.lastBody["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	assert.JSONEq(t, `[{"coinSymbol":"PONS","direction":"","strength":0,"catalyst":"","risks":[],"evidence":[],"dataGaps":[],"marketStructure":null}]`, userText)
}
