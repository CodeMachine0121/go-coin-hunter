package analysis

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
