package insight

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// coinInsightSystemPrompt is a constant so it is byte-identical on every request and therefore cacheable.
const coinInsightSystemPrompt = `你是一位加密貨幣永續合約的研究分析師。使用者會給你一枚候選幣的素材（JSON），你要為它寫一份簡短、可比較的洞察。

素材欄位：
- coinSymbol：幣種代號。
- intelligenceHeadlines：交易所公告、新上架合約、熱門排行、鏈上新幣看板等情報標題（新到舊）。
- newsHeadlines：近 3 天的新聞標題（新到舊）；以代號搜尋，可能混入同名但無關的新聞，請自行判斷相關性。
- marketStructure：一家交易所的 USDT 永續合約行情：lastPrice、priceChangeRatio24h（0.06 表示 +6%）、quoteVolumeUsd24h、fundingRate（每期，0.0001 表示 0.01%）、openInterestUsd、openInterestChangeRatio24h。缺值代表該交易所沒提供。
- filterVerdicts：規則式過濾的結果（passed / noData），附理由。
- dataGaps：系統沒能收集到的素材。

回答規則：
1. direction：只能是 bullish（看多）、bearish（看空）、neutral（中性）。證據不足或互相矛盾時用 neutral。
2. strength：1–10 的整數，代表訊號強度與可信度；素材越少、越矛盾，越低。
3. catalyst：一兩句繁體中文，說明驅動行情的主要催化劑（例如上新合約、上幣公告、資金費率極端、持倉暴增）。沒有就寫「無明確催化劑」。
4. risks：2–4 條繁體中文短句，列出主要風險。
5. evidence：2–5 條繁體中文短句，引用素材中的具體事實與數字作為依據；不得編造素材中沒有的數字。
6. dataGaps：你判斷時缺少、且會影響結論的資訊（繁體中文短句）；沒有就給空陣列。
只根據素材判斷，不要假設你知道素材以外的最新價格或消息。`

// coinInsightAnswerSchema is the shape the analyst must answer in; structured output enforces it.
var coinInsightAnswerSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"direction": map[string]any{"type": "string", "enum": []string{"bullish", "bearish", "neutral"}},
		"strength":  map[string]any{"type": "integer"},
		"catalyst":  map[string]any{"type": "string"},
		"risks":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"evidence":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"dataGaps":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"required":             []string{"direction", "strength", "catalyst", "risks", "evidence", "dataGaps"},
	"additionalProperties": false,
}

// coinInsightAnswerTokenLimit is far above a filled answer; reaching it means the answer was cut off.
const coinInsightAnswerTokenLimit = 4000

// ClaudeCoinInsightAnalystProxy asks Claude for one coin's insight in a single request.
type ClaudeCoinInsightAnalystProxy struct {
	client         anthropic.Client
	model          string
	effort         anthropic.BetaOutputConfigEffort
	requestTimeout time.Duration
}

// NewClaudeCoinInsightAnalystProxy uses the SDK's default endpoint when the base address is empty. The SDK's own
// retries are switched off: a failing service is not asked again, and each question costs exactly one call.
func NewClaudeCoinInsightAnalystProxy(
	apiKey string, baseUrl string, model string, effort string, requestTimeout time.Duration,
) *ClaudeCoinInsightAnalystProxy {
	clientOptions := []option.RequestOption{option.WithAPIKey(apiKey), option.WithMaxRetries(0)}
	if baseUrl != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(baseUrl))
	}

	return &ClaudeCoinInsightAnalystProxy{
		client:         anthropic.NewClient(clientOptions...),
		model:          model,
		effort:         anthropic.BetaOutputConfigEffort(effort),
		requestTimeout: requestTimeout,
	}
}

// AnalyzeCoin sends the material as JSON under a fixed, cached system prompt. A refusal the fallback model also
// declines, a cut-off answer or one that does not read as the agreed shape is ErrCoinInsightAnswerUnusable.
func (claudeCoinInsightAnalystProxy *ClaudeCoinInsightAnalystProxy) AnalyzeCoin(
	executionContext context.Context, material vo.CoinInsightMaterialVo,
) (vo.CoinInsightAnswerVo, error) {
	// The material is only text, numbers and times, which always encode.
	materialJson, _ := json.Marshal(newCoinInsightMaterialWire(material))

	boundedContext, releaseWait := context.WithTimeout(executionContext, claudeCoinInsightAnalystProxy.requestTimeout)
	defer releaseWait()
	message, requestError := claudeCoinInsightAnalystProxy.client.Beta.Messages.New(boundedContext, anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(claudeCoinInsightAnalystProxy.model),
		MaxTokens: coinInsightAnswerTokenLimit,
		System: []anthropic.BetaTextBlockParam{{
			Text:         coinInsightSystemPrompt,
			CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
		}},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: claudeCoinInsightAnalystProxy.effort,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: coinInsightAnswerSchema},
		},
		// A policy decline is re-served by a fallback model inside the same call.
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(string(materialJson)))},
	})
	if requestError != nil {
		return vo.CoinInsightAnswerVo{}, fmt.Errorf("ask claude for insight: %w", requestError)
	}
	if message.StopReason == anthropic.BetaStopReasonRefusal || message.StopReason == anthropic.BetaStopReasonMaxTokens {
		return vo.CoinInsightAnswerVo{}, fmt.Errorf("%w: stopped with %s", domains.ErrCoinInsightAnswerUnusable, message.StopReason)
	}

	answerText := strings.Builder{}
	for _, block := range message.Content {
		if textBlock, isText := block.AsAny().(anthropic.BetaTextBlock); isText {
			answerText.WriteString(textBlock.Text)
		}
	}
	answer := coinInsightAnswerWire{}
	if unmarshalError := json.Unmarshal([]byte(answerText.String()), &answer); unmarshalError != nil || !answer.complete() {
		return vo.CoinInsightAnswerVo{}, fmt.Errorf("%w: unreadable answer", domains.ErrCoinInsightAnswerUnusable)
	}

	return answer.toCoinInsightAnswer(), nil
}
