# Contract Traceability Matrix — 候選幣 AI 洞察（coin-insight-analysis）

Contract: PRD.md（v1.0, Finalized；含 `7456d37` 補述）
Design map: ARCH.md
Implementation: `internal/`、`cmd/server/`（branch `feature/coin-insight-analysis`，HEAD `7456d37`）
Oracle: Acceptance Criteria（42 clauses：AC 17 · BR 21 · NFR 4）

> 靜態一致性稽核（第 2 次，自零重跑）：先僅依 PRD 推導每條 oracle，再分別比對測試斷言與正式程式碼路徑；不以「整包測試全綠」為判準，未撰寫或執行任何自創情境。對應測試個別執行作佐證（`-p 1 -count=1 -timeout 120s`，含 Postgres 儲存測試）全部通過；未呼叫真實 Anthropic API。
>
> 條款 ID 沿用第 1 次稽核；PRD 新增的規則接續編為 BR-17 ~ BR-21，BR-12 併入「AI 服務位址可改」。

## Clauses

`Spec-expected` 欄為 Phase 2 僅依 PRD 文字推導的業務可觀察結果；稽核欄比對的是經 UL-MAP / ARCH 橋接後的具體產物（如 看多↔`bullish`、「尚無成功的過濾輪次」↔`ErrNoSucceededFilteringRun`/409、「找不到這個輪次」↔`ErrPipelineRunNotFound`/404、「輪次編號必須是正整數」↔`ErrInvalidPipelineRunID`/400）。

### Section 3 — Acceptance Criteria

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | 每枚保留的幣產生一份洞察並串上過濾輪次 — Given 最新一輪成功的過濾為第 7 輪，保留 PENGU 與 STRK / When 執行一輪洞察 / Then 產生 PENGU 與 STRK 兩份洞察 And 洞察輪次狀態為「成功」 And 洞察輪次記下其來源為第 7 輪過濾 | 產生且僅產生 PENGU、STRK 兩份洞察；輪次成功；來源過濾輪次＝第 7 輪 | `internal/domain/service/coin_insight_service.go:55-85,144` | `internal/application/tests/coin_insight_application_test.go:130` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | 從未有成功的過濾 — Given 從未有過成功的過濾輪次 / When 交易者手動觸發洞察 / Then 被告知「尚無成功的過濾輪次」 And 沒有建立任何洞察輪次 | 被拒絕並出現「尚無成功的過濾輪次」；洞察輪次數不變 | `coin_insight_service.go:60-62`；`internal/controller/coin_insight_controller.go:24-26` | `coin_insight_application_test.go:162`（未設 `Create` 期望，gomock 嚴格）；`internal/controller/tests/coin_insight_controller_test.go:54` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | 手動觸發的洞察輪次記為手動 — When 交易者手動觸發一輪洞察 / Then 該洞察輪次的觸發來源為「手動」 | 觸發來源＝手動 | `internal/application/coin_insight_application.go:20-22`；`coin_insight_service.go:81` | `coin_insight_application_test.go:144` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | 合法值照收 — Given AI 對 PENGU 回覆方向「看多」、強度 7 / When 執行一輪洞察 / Then PENGU 的洞察方向為看多、強度為 7 | 方向＝看多、強度＝7 | `internal/domain/models/domains/coin_insight_answer_domain.go:29-32,50` | `internal/domain/models/domains/tests/insight_domains_test.go:23`；`coin_insight_application_test.go:146` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | 強度高於 10 夾回 10 — Given AI 對 PENGU 回覆強度 12 / When 執行一輪洞察 / Then PENGU 的洞察強度為 10 | 強度＝10 | `coin_insight_answer_domain.go:50` | `insight_domains_test.go:25` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-6 | 強度低於 1 夾回 1 — Given AI 對 PENGU 回覆強度 0 / When 執行一輪洞察 / Then PENGU 的洞察強度為 1 | 強度＝1 | `coin_insight_answer_domain.go:50` | `insight_domains_test.go:26` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | 不認得的方向視為中性 — Given AI 對 PENGU 回覆方向「暴漲」 / When 執行一輪洞察 / Then PENGU 的洞察方向為中性 | 方向＝中性 | `coin_insight_answer_domain.go:29-32` | `insight_domains_test.go:27` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | 第一次不合格重問一次後成功 — Given AI 對 PENGU 第一次回覆格式不合格、第二次合格 / When 執行一輪洞察 / Then PENGU 的洞察為成功 And AI 對 PENGU 被詢問 2 次 | PENGU 成功；被詢問恰 2 次 | `coin_insight_service.go:105-108` | `coin_insight_application_test.go:182` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | 兩次都不合格即該幣分析失敗 — Given AI 對 PENGU 兩次回覆都格式不合格 And AI 對 STRK 回覆合格 / When 執行一輪洞察 / Then PENGU 記為分析失敗，原因為「AI 回覆格式不合格」 And STRK 的洞察為成功 And 洞察輪次狀態為「成功」 | PENGU 失敗、原因「AI 回覆格式不合格」；STRK 成功；輪次成功 | `coin_insight_service.go:109-116`；`domains/coin_insight_errors.go:8` | `coin_insight_application_test.go:184` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | 全部幣分析失敗即輪次失敗 — Given AI 服務對每一枚幣都出錯 / When 執行一輪洞察 / Then 洞察輪次狀態為「失敗」 And 失敗原因為「所有候選幣分析失敗」 | 輪次失敗，原因「所有候選幣分析失敗」 | `domains/pipeline_run_domain.go`（`ConcludeInsight`）；`coin_insight_service.go:144` | `coin_insight_application_test.go:220`；`insight_domains_test.go:86` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-11 | 查不到新聞仍分析 — Given STRK 查不到近期新聞 / When 執行一輪洞察 / Then STRK 的洞察為成功 And STRK 的資料缺口包含「查不到近期新聞」 | STRK 成功；缺口含「查不到近期新聞」 | `internal/domain/service/coin_insight_material_service.go:77-85` | `coin_insight_application_test.go:234` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-12 | 查不到市場結構仍分析 — Given PONS 在幣安、Bybit、OKX 都查不到永續合約市場結構 / When 執行一輪洞察 / Then PONS 的洞察為成功 And PONS 的資料缺口包含「查不到永續合約市場結構」 | PONS 成功；缺口含「查不到永續合約市場結構」 | `coin_insight_material_service.go:88-99` | `coin_insight_application_test.go:234`（整份優先序清單皆查無）；`cmd/server/dependencies_test.go`（清單恰為三家） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-13 | 超過上限只分析最早被提及的 20 枚 — Given 最新成功過濾保留 25 枚幣 / When 執行一輪洞察 / Then 只有最早被提及的 20 枚產生洞察 | 恰 20 份，且為最早被提及的 20 枚 | `domains/insight_candidate_selection_domain.go:20-50`；`coin_insight_service.go:75-76` | `coin_insight_application_test.go:256`；`insight_domains_test.go:55`（打亂輸入順序） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-14 | 最新洞察只看最新一輪成功的洞察 — Given 第 1 輪與第 2 輪洞察都成功 / When 交易者查看最新洞察 / Then 只看到第 2 輪的洞察 | 只回第 2 輪 | `coin_insight_service.go:153-171`；`persistence/pipeline_run_repository.go`（`FindLatestSucceeded`） | `coin_insight_application_test.go:370`＋`persistence/tests/discovery_repositories_test.go`（取最新成功） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-15 | 從未成功時為空 — Given 尚無成功的洞察輪次 / When 交易者查看最新洞察 / Then 看到空清單 | 空清單（非錯誤） | `coin_insight_service.go:158-160` | `coin_insight_application_test.go:384`；`coin_insight_controller_test.go:93` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-16 | 查看某一輪的結果含失敗的幣 — Given 第 2 輪洞察中 PENGU 分析失敗、STRK 成功 / When 交易者查看第 2 輪洞察的結果 / Then 看到 PENGU 為分析失敗及原因、STRK 的完整洞察 | PENGU 失敗＋原因；STRK 完整洞察 | `coin_insight_service.go:175-192`；`persistence/coin_insight_repository.go` | `coin_insight_application_test.go:394`；`persistence/tests/insight_repositories_test.go:14` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-17 | 查看不存在的輪次 — When 交易者查看一個不存在的輪次的洞察 / Then 被告知「找不到這個輪次」 | 出現「找不到這個輪次」 | `coin_insight_service.go:178-180`；`coin_insight_controller.go:54-56` | `coin_insight_controller_test.go:121`；`coin_insight_application_test.go:411` | asserts-oracle | produces-oracle | ✅ conforms |

### Section 4 — Core Business Rules & Edge Cases

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| BR-1 | 訊號方向：看多、看空、中性三者之一；其他值一律為中性。 | 方向必為三者之一；其他一律中性 | `coin_insight_answer_domain.go:29-32` | `insight_domains_test.go:23-27` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | 訊號強度：1–10 的整數；超出夾回邊界。 | 1–10 整數，超出取邊界 | `coin_insight_answer_domain.go:50` | `insight_domains_test.go:25-26` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | 素材上限：情報標題最多 20 則（探索時間窗內，新到舊） | 每枚 ≤ 20 則；只取探索時間窗內（隨探索時間窗設定而變）；新到舊 | `coin_insight_material_service.go:50-61`；`internal/config/application_config.go`（`MaximumIntelligenceHeadlines: 20`）；**`cmd/server/dependencies.go:91`（`IntelligenceWindow = Discovery.Window`）**；`persistence/coin_intelligence_repository.go`（`published_at DESC`） | `coin_insight_application_test.go:443`（25 → 20）；`config/tests/application_config_test.go:62`（預設 20）；`insight_repositories_test.go:35`（新到舊＋時間下限）；時間窗只以測試自訂 policy（72h）驗證 | shallow | produces-oracle | 🟠 mis-asserted |
| BR-4 | 素材上限：新聞標題最多 10 則（近 3 天）。 | 每枚 ≤ 10 則；僅近 3 天 | `coin_insight_material_service.go:77-80`；`application_config.go`（10、72h）；`news/google_news_coin_news_proxy.go:44-87` | `application_config_test.go:62`；`news/tests/google_news_coin_news_proxy_test.go:16`；`coin_insight_application_test.go:114` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | 市場結構來源順序：幣安 → Bybit → OKX，取第一家有該幣 USDT 永續合約的；欄位查不到的留空。 | 依 幣安→Bybit→OKX 取第一家有者；缺欄位留空 | `dependencies.go:77-83`；`coin_insight_material_service.go:88-96` | `cmd/server/dependencies_test.go:13`（順序）；`coin_insight_application_test.go:238-241`（第一個有的勝出）；`marketdata/tests/market_structure_proxies_test.go:64,106`（留空） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | 資料缺口：新聞查不到 →「查不到近期新聞」；三家都查不到市場結構 →「查不到永續合約市場結構」；另加入 AI 自行回報的缺口。來源出錯與查無資料一視同仁記為缺口，不讓本輪失敗。 | 兩條固定缺口＋AI 缺口併入；出錯＝查無，記缺口、輪次不因此失敗 | `coin_insight_material_service.go:13-16,82-99`；`coin_insight_answer_domain.go:34-43` | `coin_insight_application_test.go:234,475`；`insight_domains_test.go:40` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-7 | 重問：AI 回覆無法解讀為規定格式（或被 AI 拒答）→ 重問一次；第二次仍不行 →「AI 回覆格式不合格」。 | 無法解讀/拒答 → 再問恰一次；仍不行 → 「AI 回覆格式不合格」 | `coin_insight_service.go:105-113`；`infrastructure/insight/claude_coin_insight_analyst_proxy.go:110-123` | `coin_insight_application_test.go:182-185`；`insight/tests/claude_coin_insight_analyst_proxy_test.go:98` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-8 | AI 服務出錯（連線、額度、伺服器錯誤）→ 不重問，直接記為該幣分析失敗並寫明原因。 | 服務錯誤時只送出 1 次，該幣失敗並寫原因 | `coin_insight_service.go:106-115`；`claude_coin_insight_analyst_proxy.go:68`（`WithMaxRetries(0)`） | `coin_insight_application_test.go:186`；`claude_coin_insight_analyst_proxy_test.go:127`（503 → 恰 1 次請求、非「不可用」） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-9 | 輪次：至少一枚成功 → 成功；全部失敗 → 失敗「所有候選幣分析失敗」 | ≥1 → 成功；0 → 失敗「所有候選幣分析失敗」 | `pipeline_run_domain.go`（`ConcludeInsight`） | `insight_domains_test.go:86`；`coin_insight_application_test.go:215,228` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-10 | 輪次寫不進去即整體失敗 | 輪次無法寫入 → 整體失敗 | `coin_insight_service.go:86-88,135-147` | `coin_insight_application_test.go:326,338` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-11 | 重啟殘留執行中 → 失敗（被重啟中斷）。 | 殘留執行中 → 失敗「被重啟中斷」 | `internal/domain/service/pipeline_run_service.go:39-53`（不分步驟）；`cmd/server/main.go:38` | `application/tests/pipeline_run_application_test.go:31` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-12 | AI 設定（營運者可調整）：模型預設 Claude Opus 5.5、思考深度預設「低」（萃取型任務）；同時分析上限 3、每輪上限 20。AI 服務位址可改（例如經由代理）。 | 預設模型＝Claude Opus 5.5、思考深度＝低；可調；AI 服務位址可改 | `application_config.go:137-139`；`claude_coin_insight_analyst_proxy.go:68-71`（`WithBaseURL`） | `application_config_test.go:62`（預設 model/effort/位址空）、`:77`（effort 覆寫）；`claude_coin_insight_analyst_proxy_test.go:64`（請求送至指定位址、帶設定的 model/effort） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-13 | 同時分析上限 3、每輪上限 20。 | 同時 ≤ 3；每輪 ≤ 20 | `coin_insight_service.go:98-104`；`application_config.go`（3／20） | `coin_insight_application_test.go:256`；`application_config_test.go:62` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-14 | Flow：每枚幣收集素材（情報標題、新聞、市場結構、過濾結果） | 素材含情報、新聞、市場結構、過濾結果 | `coin_insight_material_service.go:65-75`；`insight/claude_insight_wire.go:43-80` | `coin_insight_application_test.go:152`；`claude_coin_insight_analyst_proxy_test.go:88-95` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-15 | Edge：查看某一輪洞察時輪次編號不是正整數 → 被告知「輪次編號必須是正整數」。 | 出現「輪次編號必須是正整數」 | `internal/utilities/pipeline_run_id_parameter.go:12-18`；`coin_insight_controller.go:47-50` | `coin_insight_controller_test.go:119` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-16 | Edge：AI 金鑰未設定 → 每枚幣都會分析失敗，輪次失敗並寫明原因。 | 每枚失敗；輪次失敗並寫明原因 | `application_config.go:136`；SDK 401（重試關閉）→ `coin_insight_service.go:109-116,144` | `coin_insight_application_test.go:220` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-17 | AI 拒答備援：若模型以政策理由拒答，由 Claude 服務端改派備援模型在同一次詢問中代答（不額外計次）；備援也拒答才算回覆不可用。 | 拒答時同一次詢問內由備援模型代答、不多算一次；備援也拒答 → 不可用（進入重問規則） | `claude_coin_insight_analyst_proxy.go:102-104`（`Fallbacks: default` + beta）、`:110-111`（最終 `refusal` → 不可用） | `claude_coin_insight_analyst_proxy_test.go:64`（請求帶備援設定）、`:98`（refusal → 不可用） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-18 | 每次詢問恰好一次呼叫：系統不自動重送（連線、額度、伺服器錯誤都不重送），因此每輪 AI 呼叫至多「分析上限 × 2」。 | 一次詢問＝一次呼叫；錯誤不重送；每輪 ≤ 上限×2 | `claude_coin_insight_analyst_proxy.go:68`；`coin_insight_service.go:105-108` | `claude_coin_insight_analyst_proxy_test.go:127`；`coin_insight_application_test.go:182-186,256` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-19 | 挑選順序：依探索時「最早被提及」由早到晚；查不到探索紀錄的保留幣排在最後。 | 最早被提及由早到晚；無探索紀錄者排最後 | `insight_candidate_selection_domain.go:34-48` | `insight_domains_test.go:55`（含無紀錄者置於輸入首尾兩種情況） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-20 | 市場結構來源記錄：每份洞察記下市場結構取自哪一家交易所。 | 洞察帶有市場結構來源交易所 | `coin_insight_answer_domain.go:56-58`；`entities/coin_insight.go:18` | `coin_insight_application_test.go:146`（幣安）、`:250`（Bybit） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-21 | Edge：查看某個存在但不是洞察步驟的輪次（例如過濾輪次）的洞察 → 空清單。 | 回空清單（非「找不到這個輪次」） | `coin_insight_service.go:175-192`（只驗輪次存在，不驗步驟） | `coin_insight_controller_test.go:128`「a known run」（輪次未設步驟、只斷言 200，未斷言回應為空清單）；無任何測試以過濾輪次查詢 | shallow | produces-oracle | 🟠 mis-asserted |

### Section 6 — Non-Functional Requirements

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| NFR-1 | Performance：每次 AI 詢問逾時 120 秒 | 單次詢問逾 120 秒即放棄 | `application_config.go`（120s）；`claude_coin_insight_analyst_proxy.go:89` | `application_config_test.go:62`（預設 120s）；`claude_coin_insight_analyst_proxy_test.go:146`（逾時即放棄） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-2 | Performance：素材收集每來源逾時 15 秒。 | 每來源逾 15 秒即放棄（並記缺口） | `coin_insight_material_service.go:77,89`；`application_config.go:147` | `application_config_test.go:62`（15s）；`coin_insight_application_test.go:475`（慢來源被切斷並記缺口） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-3 | Cost：資料來源全部免費；AI 呼叫每輪最多 20 枚 × 2 次；固定的系統提示可被快取以降低費用。 | 每輪呼叫 ≤ 40；系統提示固定且可快取 | `claude_coin_insight_analyst_proxy.go:17-35,68,94-97`；`coin_insight_service.go:105-108` | `claude_coin_insight_analyst_proxy_test.go:85-86,127`；`coin_insight_application_test.go:256` | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-4 | Security：AI 金鑰只從環境變數讀取，不寫入資料庫、不出現在輪次或洞察內容中。 | 金鑰只來自環境變數；資料庫／輪次／洞察皆不含金鑰 | `application_config.go:136`；`entities/coin_insight.go`（無金鑰欄位）；失敗原因僅來自錯誤字串 | `application_config_test.go:77`；`claude_coin_insight_analyst_proxy_test.go:127`（錯誤字串不含金鑰） | asserts-oracle | produces-oracle | ✅ conforms |

### 判定說明（非一致條款）

- **BR-3 🟠**：「每枚 ≤ 20 則」與「新到舊」現已釘住。但「探索時間窗內」這條綁定規則只存在於 `cmd/server/dependencies.go:91`（`IntelligenceWindow: applicationConfig.Discovery.Window`），沒有任何測試釘住。應用層測試以自訂 policy 的 72h 驗證，而探索時間窗預設剛好也是 72h（與新聞回看 `NewsLookback` 相同）。所以這一行若改成 `Insight.NewsLookback` 或寫死 72h，所有測試仍綠，但營運者調整 `DISCOVERY_WINDOW_HOURS` 時就會偏離規則。
- **BR-21 🟠**：程式碼正確（`GetCoinInsightsOfPipelineRun` 只確認輪次存在就回該輪洞察，過濾輪次自然為空清單）。但沒有測試建立「過濾步驟的輪次」來查詢；`coin_insight_controller_test.go:128` 只斷言 HTTP 200，不檢查回應內容是否為空清單。

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| `internal/config/application_config.go`（`GOOGLE_NEWS_BASE_URL`）→ `dependencies.go`（`news.NewGoogleNewsCoinNewsProxy`） | 新聞搜尋位址可覆寫；README:75 有寫，PRD 只寫「AI 服務位址可改」 | undocumented（良性） |

Out of Scope 檢核（投資裁決、獵捕結果表、排程、AI 多輪對話或自行查資料、付費資料來源）：`internal/job/` 無洞察 job；每枚幣單次請求、無 tool use；備援模型屬同一次請求，不是多輪對話；素材來源皆為免費公開端點。**無 out-of-scope 違規。**

前次孤兒的處置：SDK 自動重試已關閉（BR-18）；伺服器端備援（BR-17）、無探索紀錄者排序（BR-19）、市場結構來源交易所（BR-20）、AI 服務位址（BR-12）、非洞察輪次回空清單（BR-21）皆已納入 PRD；`.env.example:18` 的措辭已更正；UL-MAP 的技術名稱已補上。

附註（非合約條款）：`cmd/server/dependencies_test.go` 是 `package main` 白箱測試，與 `.claude/rules/testing.md`「一律外部黑箱、放 `tests/`」的放置規則有出入。組裝根本來沒有外部可 import 的入口，這可能是刻意的例外，但規則檔未記載。

## Summary

- Conforms: 40/42 clauses ✅ (95.2%)
- Violations: —
- Mis-asserted: BR-3, BR-21（green test asserts a weaker thing）
- Partial: —
- Gaps: —
- Unclear: —
- Orphans: 1（良性；0 件屬 Out of Scope 違規）

---

## Resolution（第二輪稽核後處理，2026-10-05）

| Clause | 原狀態 | 處理 | 驗證 |
| :--- | :--- | :--- | :--- |
| BR-3 | 🟠 mis-asserted | 洞察政策改由組裝根的 `coinInsightPolicyFor` 單一處組出，情報時間窗取自探索時間窗；以非預設的 24 小時探索窗測試（commit `273d811`） | 把時間窗改接新聞回看期的 mutation 被抓到 |
| BR-21 | 🟠 mis-asserted | 新增「其他步驟的輪次查洞察 → 空清單」路由測試，斷言回應為 `[]` | — |
| Orphan（新聞搜尋位址可改） | ⚠️ | PRD 已註明 AI 服務位址與新聞搜尋位址皆可改 | — |
| 組裝根白箱測試 | 備註 | `.claude/rules/testing.md` 已記錄組裝根為唯一同 package 測試的例外 | — |

處理後：**42 conforms · 0 violation · 0 mis-asserted · 0 partial · 0 unclear · 0 orphan**。全套測試（含 PostgreSQL 儲存層）通過，`internal/` 非 mock 程式碼覆蓋率 100%。
