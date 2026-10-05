package verdict

import (
	"bytes"
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

// huntVerdictSystemPrompt is a constant so it is byte-identical on every request and therefore cacheable.
const huntVerdictSystemPrompt = `你是一個加密貨幣永續合約交易團隊的投資長（CIO）。研究員已為本輪每一枚候選幣寫好洞察；你要一次看完全部，為每一枚幣做出可執行的裁決。

素材是一個 JSON 陣列，每一枚幣包含：
- coinSymbol；direction（研究員判斷：bullish / bearish / neutral）、strength（1–10）、catalyst、risks、evidence、dataGaps；
- marketStructure：此刻一家交易所的 USDT 永續合約行情（lastPrice、priceChangeRatio24h、quoteVolumeUsd24h、fundingRate、openInterestUsd、openInterestChangeRatio24h，缺值代表查不到）；可能為 null。

為每一枚幣回答：
1. action：long（做多）、short（做空）、watch（觀望）、avoid（避開）。證據薄弱、資料缺口大、或洞察與市場結構互相矛盾且無法取捨時，用 watch；風險明顯大於機會時，用 avoid。
2. confidence：0–100 的整數，代表你對這個裁決的把握。
3. leverage：建議槓桿倍數（整數，1–5；watch / avoid 填 0）。
4. positionSizePercent：建議部位佔總資金的百分比（0–10；watch / avoid 填 0）。把本輪所有 long / short 一起考慮，總和不宜過高，越不確定越小。
5. stopLossPercent：停損距離，佔目前價格的百分比（例如 8 表示 8%）；takeProfitPercent：停利距離，同樣以百分比表示。watch / avoid 填 0。不要給價格，系統會依方向換算。
6. rationale：兩三句繁體中文，說明裁決理由，引用素材中的具體事實。
7. conflictResolution：一兩句繁體中文，說明研究員的洞察之間、或洞察與此刻市場結構矛盾時，你採信哪一邊、為什麼；沒有矛盾就寫「無明顯矛盾」。

規則：每一枚素材中的幣都要回答一次，不要遺漏，也不要加入素材以外的幣；只根據素材判斷，不要假設你知道素材以外的價格或消息；本系統不下單，你的裁決只供交易者參考。`

var huntVerdictAnswerSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"verdicts": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"coinSymbol":          map[string]any{"type": "string"},
					"action":              map[string]any{"type": "string", "enum": []string{"long", "short", "watch", "avoid"}},
					"confidence":          map[string]any{"type": "integer"},
					"leverage":            map[string]any{"type": "integer"},
					"positionSizePercent": map[string]any{"type": "number"},
					"stopLossPercent":     map[string]any{"type": "number"},
					"takeProfitPercent":   map[string]any{"type": "number"},
					"rationale":           map[string]any{"type": "string"},
					"conflictResolution":  map[string]any{"type": "string"},
				},
				"required": []string{"coinSymbol", "action", "confidence", "leverage", "positionSizePercent",
					"stopLossPercent", "takeProfitPercent", "rationale", "conflictResolution"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"verdicts"},
	"additionalProperties": false,
}

// huntVerdictAnswerTokenLimit leaves room for twenty coins' verdicts; reaching it means the answer was cut off.
const huntVerdictAnswerTokenLimit = 16000

// ClaudeHuntVerdictStrategistProxy asks Claude, as the chief investment officer, for every coin's verdict in one request.
type ClaudeHuntVerdictStrategistProxy struct {
	client         anthropic.Client
	model          string
	effort         anthropic.BetaOutputConfigEffort
	requestTimeout time.Duration
}

// NewClaudeHuntVerdictStrategistProxy uses the SDK's default endpoint when the base address is empty. The SDK's own
// retries are switched off: a failing service is not asked again, and each question costs exactly one call.
func NewClaudeHuntVerdictStrategistProxy(
	apiKey string, baseUrl string, model string, effort string, requestTimeout time.Duration,
) *ClaudeHuntVerdictStrategistProxy {
	clientOptions := []option.RequestOption{option.WithAPIKey(apiKey), option.WithMaxRetries(0)}
	if baseUrl != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(baseUrl))
	}

	return &ClaudeHuntVerdictStrategistProxy{
		client:         anthropic.NewClient(clientOptions...),
		model:          model,
		effort:         anthropic.BetaOutputConfigEffort(effort),
		requestTimeout: requestTimeout,
	}
}

// SynthesizeVerdicts sends every coin as one JSON array. A refusal the fallback model also declines, a cut-off answer
// or one that does not read as the agreed shape is ErrHuntVerdictAnswerUnusable.
func (claudeHuntVerdictStrategistProxy *ClaudeHuntVerdictStrategistProxy) SynthesizeVerdicts(
	executionContext context.Context, materials []vo.HuntVerdictMaterialVo,
) ([]vo.HuntVerdictAnswerVo, error) {
	// The material is only text, numbers and times, which always encode.
	materialJson, _ := json.Marshal(newHuntVerdictMaterialWires(materials))

	boundedContext, releaseWait := context.WithTimeout(executionContext, claudeHuntVerdictStrategistProxy.requestTimeout)
	defer releaseWait()
	message, requestError := claudeHuntVerdictStrategistProxy.client.Beta.Messages.New(boundedContext, anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(claudeHuntVerdictStrategistProxy.model),
		MaxTokens: huntVerdictAnswerTokenLimit,
		System: []anthropic.BetaTextBlockParam{{
			Text:         huntVerdictSystemPrompt,
			CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
		}},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: claudeHuntVerdictStrategistProxy.effort,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: huntVerdictAnswerSchema},
		},
		// A policy decline is re-served by a fallback model inside the same call.
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(string(materialJson)))},
	})
	if requestError != nil {
		return nil, fmt.Errorf("ask claude for verdicts: %w", requestError)
	}
	if message.StopReason == anthropic.BetaStopReasonRefusal || message.StopReason == anthropic.BetaStopReasonMaxTokens {
		return nil, fmt.Errorf("%w: stopped with %s", domains.ErrHuntVerdictAnswerUnusable, message.StopReason)
	}

	answerText := strings.Builder{}
	for _, block := range message.Content {
		if textBlock, isText := block.AsAny().(anthropic.BetaTextBlock); isText {
			answerText.WriteString(textBlock.Text)
		}
	}
	answer := huntVerdictAnswerWire{}
	decoder := json.NewDecoder(bytes.NewReader([]byte(answerText.String())))
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
