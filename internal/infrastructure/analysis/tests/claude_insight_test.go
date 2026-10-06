package analysis_test

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
	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/analysis"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMessagesApi answers every Messages request with the given status and assistant text, remembering the last request.
type fakeMessagesApi struct {
	lastBody   map[string]any
	lastHeader http.Header
	lastPath   string
}

func (fakeApi *fakeMessagesApi) serve(t *testing.T, statusCode int, stopReason string, answerText string) string {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		fakeApi.lastBody = map[string]any{}
		_ = json.Unmarshal(body, &fakeApi.lastBody)
		fakeApi.lastHeader = request.Header.Clone()
		fakeApi.lastPath = request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(statusCode)
		if statusCode != http.StatusOK {
			_, _ = writer.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"bad request"}}`))
			return
		}
		textJson, _ := json.Marshal(answerText)
		_, _ = writer.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","content":[{"type":"text","text":` +
			string(textJson) + `}],"stop_reason":"` + stopReason + `","usage":{"input_tokens":10,"output_tokens":10}}`))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

func pengu() vo.CoinInsightMaterialVo {
	fundingRate := decimal.RequireFromString("0.0001")
	fundingIntervalHours := 4
	return vo.CoinInsightMaterialVo{
		CoinSymbol:            "PENGU",
		IntelligenceHeadlines: []vo.HeadlineVo{{SourceName: "binanceAnnouncement", Title: "Binance Futures Will Launch PENGUUSDT", PublishedAt: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)}},
		NewsHeadlines:         []vo.HeadlineVo{{SourceName: "CoinDesk", Title: "PENGU rallies"}},
		MarketStructure:       &vo.PerpetualMarketStructureVo{ExchangeName: "幣安", FundingRate: &fundingRate, FundingIntervalHours: &fundingIntervalHours},
		FilterVerdicts:        []vo.FilterVerdictVo{{FilterName: "unlockSchedule", Outcome: vo.FilterOutcomeNoData, Reason: "查不到解鎖時程"}},
		DataGaps:              []string{"查不到近期新聞"},
	}
}

const readableAnswer = `{"direction":"bullish","strength":7,"catalyst":"幣安上新永續合約","risks":["解鎖"],"evidence":["資金費率 0.01%"],"dataGaps":[]}`

func TestClaudeAnalystReadsAnAnswerAndAsksInTheAgreedShape(t *testing.T) {
	fakeApi := &fakeMessagesApi{}
	proxy := newInsightProxy("test-key", fakeApi.serve(t, http.StatusOK, "end_turn", readableAnswer),
		"claude-opus-5-5", "low", 5*time.Second)

	answer, analyzeError := proxy.AnalyzeCoin(context.Background(), pengu())

	require.NoError(t, analyzeError)
	assert.Equal(t, vo.CoinInsightAnswerVo{Direction: "bullish", Strength: 7, Catalyst: "幣安上新永續合約", Risks: []string{"解鎖"},
		Evidence: []string{"資金費率 0.01%"}, DataGaps: []string{}}, answer)

	assert.Equal(t, "/v1/messages", fakeApi.lastPath)
	assert.Equal(t, "test-key", fakeApi.lastHeader.Get("X-Api-Key"))
	assert.Contains(t, fakeApi.lastHeader.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01")
	assert.Equal(t, "claude-opus-5-5", fakeApi.lastBody["model"])
	assert.Equal(t, "default", fakeApi.lastBody["fallbacks"])
	outputConfig := fakeApi.lastBody["output_config"].(map[string]any)
	assert.Equal(t, "low", outputConfig["effort"])
	format := outputConfig["format"].(map[string]any)
	assert.Equal(t, "json_schema", format["type"])
	assert.ElementsMatch(t, []any{"direction", "strength", "catalyst", "risks", "evidence", "dataGaps"}, format["schema"].(map[string]any)["required"])
	system := fakeApi.lastBody["system"].([]any)[0].(map[string]any)
	assert.Equal(t, "ephemeral", system["cache_control"].(map[string]any)["type"])

	userText := fakeApi.lastBody["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	material := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(userText), &material))
	assert.Equal(t, "PENGU", material["coinSymbol"])
	assert.Equal(t, "0.0001", material["marketStructure"].(map[string]any)["fundingRate"])
	assert.Equal(t, float64(4), material["marketStructure"].(map[string]any)["fundingIntervalHours"])
	assert.Nil(t, material["marketStructure"].(map[string]any)["lastPrice"])
	assert.Equal(t, []any{"查不到近期新聞"}, material["dataGaps"])
	assert.Equal(t, "unlockSchedule", material["filterVerdicts"].([]any)[0].(map[string]any)["rule"])
}

func TestClaudeAnalystTellsUnusableAnswersFromServiceErrors(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		stopReason   string
		answerText   string
		wantUnusable bool
	}{
		{name: "a refusal, even one carrying a readable answer", statusCode: http.StatusOK, stopReason: "refusal", answerText: readableAnswer, wantUnusable: true},
		{name: "a cut-off answer", statusCode: http.StatusOK, stopReason: "max_tokens", answerText: `{"direction":"bull`, wantUnusable: true},
		{name: "text that is not json", statusCode: http.StatusOK, stopReason: "end_turn", answerText: "看多", wantUnusable: true},
		{name: "a missing field", statusCode: http.StatusOK, stopReason: "end_turn", answerText: `{"direction":"bullish","strength":7}`, wantUnusable: true},
		{name: "a rejected request", statusCode: http.StatusBadRequest, wantUnusable: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeApi := &fakeMessagesApi{}
			proxy := newInsightProxy("test-key", fakeApi.serve(t, testCase.statusCode, testCase.stopReason, testCase.answerText),
				"claude-opus-5-5", "low", 5*time.Second)

			_, analyzeError := proxy.AnalyzeCoin(context.Background(), vo.CoinInsightMaterialVo{CoinSymbol: "PENGU"})

			require.Error(t, analyzeError)
			assert.Equal(t, testCase.wantUnusable, errors.Is(analyzeError, domains.ErrCoinInsightAnswerUnusable))
		})
	}
}

func TestClaudeAnalystMakesExactlyOneCallPerQuestionAndKeepsTheKeyOutOfErrors(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requestCount++
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte(`{"type":"error","error":{"type":"overloaded_error","message":"overloaded"}}`))
	}))
	t.Cleanup(server.Close)
	proxy := analysis.NewClaudeAnalysisProxy("sk-ant-secret-test-key", server.URL, analysis.ClaudeModelSettings{Model: "claude-opus-5-5", Effort: "low", RequestTimeout: 5 * time.Second}, analysis.ClaudeModelSettings{})

	_, analyzeError := proxy.AnalyzeCoin(context.Background(), vo.CoinInsightMaterialVo{CoinSymbol: "PENGU"})

	require.Error(t, analyzeError)
	assert.Equal(t, 1, requestCount)
	assert.False(t, errors.Is(analyzeError, domains.ErrCoinInsightAnswerUnusable))
	assert.NotContains(t, analyzeError.Error(), "sk-ant-secret-test-key")
}

func TestClaudeAnalystGivesUpAtItsDeadline(t *testing.T) {
	released := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-released }))
	t.Cleanup(func() {
		close(released)
		server.Close()
	})
	proxy := analysis.NewClaudeAnalysisProxy("test-key", server.URL, analysis.ClaudeModelSettings{Model: "claude-opus-5-5", Effort: "low", RequestTimeout: 100 * time.Millisecond}, analysis.ClaudeModelSettings{})
	startedAt := time.Now()

	_, analyzeError := proxy.AnalyzeCoin(context.Background(), vo.CoinInsightMaterialVo{CoinSymbol: "PENGU"})

	require.Error(t, analyzeError)
	assert.ErrorIs(t, analyzeError, context.DeadlineExceeded)
	assert.Less(t, time.Since(startedAt), time.Second)
}

func TestClaudeAnalystUsesTheDefaultEndpointWithoutABaseAddress(t *testing.T) {
	assert.NotNil(t, analysis.NewClaudeAnalysisProxy("test-key", "", analysis.ClaudeModelSettings{Model: "claude-opus-5-5", Effort: "low", RequestTimeout: time.Second}, analysis.ClaudeModelSettings{}))
}

// newInsightProxy builds the analysis proxy with only the insight capability configured.
func newInsightProxy(apiKey string, baseUrl string, model string, effort string, requestTimeout time.Duration) *analysis.ClaudeAnalysisProxy {
	return analysis.NewClaudeAnalysisProxy(apiKey, baseUrl, analysis.ClaudeModelSettings{Model: model, Effort: effort, RequestTimeout: requestTimeout}, analysis.ClaudeModelSettings{})
}
