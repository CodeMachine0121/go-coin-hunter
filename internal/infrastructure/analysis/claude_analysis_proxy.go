package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// errClaudeAnswerUnusable is Claude answering in a way that cannot be read; each capability reports it as its own domain error.
var errClaudeAnswerUnusable = errors.New("claude answer unusable")

// ClaudeModelSettings is how one capability asks Claude: which model, how hard it thinks, and how long it may take.
type ClaudeModelSettings struct {
	Model          string
	Effort         string
	RequestTimeout time.Duration
}

// ClaudeAnalysisProxy is the one way out to Claude: the per-coin insight analyst and the round's chief investment officer.
// Every request uses structured output under a cached system prompt, a server-side fallback for policy declines, and no
// automatic resending, so each question costs exactly one call.
type ClaudeAnalysisProxy struct {
	client          anthropic.Client
	insightSettings ClaudeModelSettings
	verdictSettings ClaudeModelSettings
}

// NewClaudeAnalysisProxy uses the SDK's default endpoint when the base address is empty.
func NewClaudeAnalysisProxy(
	apiKey string, baseUrl string, insightSettings ClaudeModelSettings, verdictSettings ClaudeModelSettings,
) *ClaudeAnalysisProxy {
	clientOptions := []option.RequestOption{option.WithAPIKey(apiKey), option.WithMaxRetries(0)}
	if baseUrl != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(baseUrl))
	}

	return &ClaudeAnalysisProxy{
		client:          anthropic.NewClient(clientOptions...),
		insightSettings: insightSettings,
		verdictSettings: verdictSettings,
	}
}

// AnalyzeCoin asks for one coin's insight; an unreadable answer is ErrCoinInsightAnswerUnusable.
func (claudeAnalysisProxy *ClaudeAnalysisProxy) AnalyzeCoin(
	executionContext context.Context, material vo.CoinInsightMaterialVo,
) (vo.CoinInsightAnswerVo, error) {
	// The material is only text, numbers and times, which always encode.
	materialJson, _ := json.Marshal(newCoinInsightMaterialWire(material))
	answerText, askError := claudeAnalysisProxy.askForJson(executionContext, claudeAnalysisProxy.insightSettings,
		coinInsightSystemPrompt, coinInsightAnswerSchema, coinInsightAnswerTokenLimit, string(materialJson))
	if errors.Is(askError, errClaudeAnswerUnusable) {
		return vo.CoinInsightAnswerVo{}, fmt.Errorf("%w: %v", domains.ErrCoinInsightAnswerUnusable, askError)
	}
	if askError != nil {
		return vo.CoinInsightAnswerVo{}, fmt.Errorf("ask claude for insight: %w", askError)
	}

	answer := coinInsightAnswerWire{}
	if unmarshalError := json.Unmarshal([]byte(answerText), &answer); unmarshalError != nil || !answer.complete() {
		return vo.CoinInsightAnswerVo{}, fmt.Errorf("%w: unreadable answer", domains.ErrCoinInsightAnswerUnusable)
	}

	return answer.toCoinInsightAnswer(), nil
}

// SynthesizeVerdicts asks for every coin's verdict as one JSON array; an unreadable answer is ErrHuntVerdictAnswerUnusable.
func (claudeAnalysisProxy *ClaudeAnalysisProxy) SynthesizeVerdicts(
	executionContext context.Context, materials []vo.HuntVerdictMaterialVo,
) ([]vo.HuntVerdictAnswerVo, error) {
	// The material is only text, numbers and times, which always encode.
	materialJson, _ := json.Marshal(newHuntVerdictMaterialWires(materials))
	answerText, askError := claudeAnalysisProxy.askForJson(executionContext, claudeAnalysisProxy.verdictSettings,
		huntVerdictSystemPrompt, huntVerdictAnswerSchema, huntVerdictAnswerTokenLimit, string(materialJson))
	if errors.Is(askError, errClaudeAnswerUnusable) {
		return nil, fmt.Errorf("%w: %v", domains.ErrHuntVerdictAnswerUnusable, askError)
	}
	if askError != nil {
		return nil, fmt.Errorf("ask claude for verdicts: %w", askError)
	}

	answer := huntVerdictAnswerWire{}
	decoder := json.NewDecoder(bytes.NewReader([]byte(answerText)))
	decoder.UseNumber()
	if decodeError := decoder.Decode(&answer); decodeError != nil || answer.Verdicts == nil {
		return nil, fmt.Errorf("%w: unreadable answer", domains.ErrHuntVerdictAnswerUnusable)
	}
	huntVerdictAnswers := make([]vo.HuntVerdictAnswerVo, 0, len(*answer.Verdicts))
	for _, verdict := range *answer.Verdicts {
		huntVerdictAnswer, readable := verdict.toHuntVerdictAnswer()
		if !readable {
			return nil, fmt.Errorf("%w: unreadable verdict", domains.ErrHuntVerdictAnswerUnusable)
		}
		huntVerdictAnswers = append(huntVerdictAnswers, huntVerdictAnswer)
	}

	return huntVerdictAnswers, nil
}

// askForJson sends one question and returns the answer text; a refusal the fallback also declines or a cut-off answer is
// errClaudeAnswerUnusable.
func (claudeAnalysisProxy *ClaudeAnalysisProxy) askForJson(
	executionContext context.Context, settings ClaudeModelSettings, systemPrompt string, answerSchema map[string]any,
	answerTokenLimit int64, question string,
) (string, error) {
	boundedContext, releaseWait := context.WithTimeout(executionContext, settings.RequestTimeout)
	defer releaseWait()
	message, requestError := claudeAnalysisProxy.client.Beta.Messages.New(boundedContext, anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(settings.Model),
		MaxTokens: answerTokenLimit,
		System: []anthropic.BetaTextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
		}},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffort(settings.Effort),
			Format: anthropic.BetaJSONOutputFormatParam{Schema: answerSchema},
		},
		// A policy decline is re-served by a fallback model inside the same call.
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(question))},
	})
	if requestError != nil {
		return "", requestError
	}
	if message.StopReason == anthropic.BetaStopReasonRefusal || message.StopReason == anthropic.BetaStopReasonMaxTokens {
		return "", fmt.Errorf("%w: stopped with %s", errClaudeAnswerUnusable, message.StopReason)
	}

	answerText := strings.Builder{}
	for _, block := range message.Content {
		if textBlock, isText := block.AsAny().(anthropic.BetaTextBlock); isText {
			answerText.WriteString(textBlock.Text)
		}
	}

	return answerText.String(), nil
}
