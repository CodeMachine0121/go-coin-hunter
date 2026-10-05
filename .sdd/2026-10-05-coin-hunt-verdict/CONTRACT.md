# Contract Traceability Matrix — 獵捕裁決與獵捕結果表（coin-hunt-verdict）

Contract: PRD.md（v1.0, Finalized）
Design map: ARCH.md（§6 Traceability）
Implementation: `internal/`（branch `feature/coin-hunt-verdict`）
Oracle: Acceptance Criteria（37 clauses：AC 17 · BR 17 · NFR 3）
Audit date: 2026-10-05

> **天花板：** 本矩陣為**靜態一致性稽核**——依 PRD 推導的預期結果（oracle），分別獨立判斷「測試是否斷言 oracle」與「程式碼路徑是否產生 oracle」。不撰寫新探針、不執行自行發明的情境。僅執行已對應到條款的既有測試作為佐證（domain / application / controller / analysis / config / cmd/server / persistence 中與裁決相關者，全數通過）；判定依 oracle 比對，非依通過與否。

**Out of Scope（負面檢查清單）：** 下單、推播、排程、績效回測。→ 掃描結果：`backgroundJobsFor` 回傳空清單（`cmd/server/dependencies.go:225-230`），無下單 / 推播 / 回測程式碼；**無越界**。

## Clauses

`Spec-expected` 欄為 Phase 2 僅依 PRD 文字推導的業務可觀察 oracle；稽核欄檢查的是經 UL-MAP / ARCH 橋接後的具體產物。

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | Scenario: 空表寫入本輪裁決 — Given 獵捕結果表為空；And 本輪裁決的幣為 BTC、ETH、BNB；When 執行一輪裁決；Then 獵捕結果表有 BTC、ETH、BNB 三列；And 三列的計算時間皆為本輪裁決時間 | 表恰好 BTC、ETH、BNB 三列，每列計算時間＝本輪裁決時間 | `hunt_verdict_service.go:121-126`；`coin_verdict.go:47`；`hunt_board_repository.go:21-51` | `verdict_repositories_test.go:31`（空表首寫 3 列）；`hunt_verdict_application_test.go:136`（每列 CalculatedAt＝本輪開始時間） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | Scenario: 本輪沒有的幣從表上移除 — Given 獵捕結果表有 BTC、ETH、BNB；And 本輪裁決的幣為 BTC、ETH；When 執行一輪裁決；Then 獵捕結果表只有 BTC、ETH 兩列；And BTC、ETH 為本輪的裁決內容與計算時間 | 表只剩 BTC、ETH；兩列內容與計算時間皆為本輪；BNB 被移除 | `hunt_board_repository.go:28-44` | `verdict_repositories_test.go:31`（Len 2、NotContains BNB、BTC/ETH 時間/內容/輪次為第二輪、ID 保留） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | Scenario: 失敗的裁決不改動結果表 — Given 獵捕結果表有 BTC；And CIO 兩次回覆都格式不合格；When 執行一輪裁決；Then 裁決輪次狀態為「失敗」；And 失敗原因為「AI 回覆格式不合格」；And 獵捕結果表仍只有原本的 BTC | 輪次失敗；原因恰為「AI 回覆格式不合格」；結果表不被改寫 | `hunt_verdict_service.go:93-114`；`hunt_verdict_errors.go:8` | `hunt_verdict_application_test.go:169`「two unreadable answers」（Times(2)、狀態失敗、原因、Rewrite 未呼叫、未存紀錄） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | Scenario: 槓桿與部位超出上限被夾回 — Given CIO 對 PENGU 給出做多、槓桿 20 倍、部位 30%；When 執行一輪裁決；Then PENGU 的建議槓桿為 5 倍、部位大小為 10% | 槓桿 5、部位 10% | `hunt_verdicts_domain.go:88-89`；`dependencies.go:126-136` | `hunt_verdicts_domain_test.go:44`（Leverage 5、PositionSizeRatio "0.1"） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | Scenario: 不認得的操作視為觀望 — Given CIO 對 PENGU 給出操作「梭哈」；When 執行一輪裁決；Then PENGU 的操作為觀望；And 建議槓桿為 0、部位大小為 0% | 操作＝觀望；槓桿 0；部位 0% | `hunt_verdicts_domain.go:47-48,59-66` | `hunt_verdicts_domain_test.go:88`「an unknown action」 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-6 | Scenario: 信心超出 100 被夾回 — Given CIO 對 PENGU 給出信心 130；When 執行一輪裁決；Then PENGU 的信心為 100 | 信心 100 | `hunt_verdicts_domain.go:56` | `hunt_verdicts_domain_test.go:44` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | Scenario: 做多的停損在下、停利在上 — Given PENGU 最新價格為 0.01；And CIO 對 PENGU 給出做多、停損距離 10%、停利距離 30%；When 執行一輪裁決；Then PENGU 的停損價為 0.009、停利價為 0.013 | 停損價恰為 0.009、停利價恰為 0.013 | `hunt_verdicts_domain.go:78-81` | `hunt_verdicts_domain_test.go:63` case 1（字串精確比對） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | Scenario: 做空的停損在上、停利在下 — Given PENGU 最新價格為 0.01；And CIO 對 PENGU 給出做空、停損距離 10%、停利距離 30%；When 執行一輪裁決；Then PENGU 的停損價為 0.011、停利價為 0.007 | 停損價 0.011、停利價 0.007 | `hunt_verdicts_domain.go:82-85` | `hunt_verdicts_domain_test.go:63` case 2 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | Scenario: 停損距離過小被夾到 1% — Given PENGU 最新價格為 0.01；And CIO 對 PENGU 給出做多、停損距離 0.5%；When 執行一輪裁決；Then PENGU 的停損距離為 1%、停損價為 0.0099 | 停損距離 1%；停損價 0.0099 | `hunt_verdicts_domain.go:78,80` | `hunt_verdicts_domain_test.go:63` case 3（ratio "0.01"、price "0.0099"） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | Scenario: 查不到最新價格的做多改判觀望 — Given PONS 查不到最新價格；And CIO 對 PONS 給出做多；When 執行一輪裁決；Then PONS 的操作為觀望；And 理由為「查不到最新價格，無法設定停損」 | 操作＝觀望；理由恰為「查不到最新價格，無法設定停損」 | `hunt_verdicts_domain.go:68-76`；`perpetual_market_structure_service.go:28-41` | `hunt_verdict_application_test.go:210`（PONS 各交易所皆無 → 觀望＋理由）；`hunt_verdicts_domain_test.go:88` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-11 | Scenario: CIO 漏掉的幣記為觀望 — Given 洞察有 PENGU 與 STRK；And CIO 只對 PENGU 給出裁決；When 執行一輪裁決；Then STRK 的操作為觀望；And 理由為「CIO 未給出裁決」 | STRK 觀望；理由「CIO 未給出裁決」 | `hunt_verdicts_domain.go:49-54` | `hunt_verdicts_domain_test.go:122`（整筆 entity 等值比對） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-12 | Scenario: CIO 多給的幣被忽略 — Given 洞察只有 PENGU；And CIO 另外對 DOGE 給出裁決；When 執行一輪裁決；Then 獵捕結果表沒有 DOGE | DOGE 不出現在結果表 | `hunt_verdicts_domain.go:45-46`（只依素材迭代）；`hunt_verdict_service.go:121-124` | `hunt_verdicts_domain_test.go:122`（多給 doge → 結果恰 2 筆、無 DOGE）；`hunt_verdict_application_test.go:136`（結果表列恰等於裁決列） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-13 | Scenario: 從未有成功的洞察 — Given 從未有過成功的洞察輪次；When 交易者手動觸發裁決；Then 被告知「尚無成功的洞察輪次」；And 沒有建立任何裁決輪次 | 被告知「尚無成功的洞察輪次」；不建立任何輪次 | `hunt_verdict_service.go:54-61`；`hunt_verdict_controller.go:24-27` | `hunt_verdict_application_test.go:228`（訊息；gomock 未設 Create 期望，若建輪次即失敗）；`hunt_verdict_controller_test.go:54` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-14 | Scenario: 成功的裁決輪次串上洞察輪次 — Given 最新一輪成功的洞察為第 9 輪；When 執行一輪裁決；Then 裁決輪次狀態為「成功」；And 裁決輪次記下其來源為第 9 輪洞察；And 觸發來源為「手動」 | 狀態成功；來源＝第 9 輪洞察；觸發＝手動 | `hunt_verdict_service.go:68-74,137-142`；`hunt_verdict_application.go:20-22` | `hunt_verdict_application_test.go:136`（Status、Step、TriggerSource、TriggeredByPipelineRunID＝9） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-15 | Scenario: 結果表依信心由高到低 — Given 獵捕結果表有信心 40 的 STRK 與信心 80 的 PENGU；When 交易者查看獵捕結果表；Then PENGU 排在 STRK 之前 | PENGU 在 STRK 之前 | `hunt_board_repository.go:53-61` | `verdict_repositories_test.go:60`（真實 DB：AAA、PENGU、STRK） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-16 | Scenario: 查看某一輪的裁決 — Given 第 3 輪裁決了 PENGU 與 STRK；When 交易者查看第 3 輪的裁決；Then 看到 PENGU 與 STRK 的裁決 | 看到 PENGU 與 STRK 兩筆裁決 | `hunt_verdict_service.go:160-177`；`coin_verdict_repository.go:30-41` | `verdict_repositories_test.go:100`（只回第 3 輪的兩筆）；`hunt_verdict_application_test.go:330`「a run's verdicts」 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-17 | Scenario: 查看不存在的輪次 — When 交易者查看一個不存在的輪次的裁決；Then 被告知「找不到這個輪次」 | 被告知「找不到這個輪次」 | `hunt_verdict_service.go:163-165`；`hunt_verdict_controller.go:54-57`；`pipeline_run_errors.go:6` | `hunt_verdict_controller_test.go:131`「an unknown run」（404＋訊息） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-1 | **操作：** 做多 `long`、做空 `short`、觀望 `watch`、避開 `avoid`；其他一律觀望。 | 四種操作原樣保留（不分大小寫/空白），其他值一律觀望 | `hunt_verdicts_domain.go:59-66,87` | `hunt_verdicts_domain_test.go:88`（梭哈→watch、" AVOID "→avoid、watch）；`:44`、`:63`（long/short） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | **數值範圍：** 信心 0–100（皆夾回邊界） | 信心 <0 → 0；>100 → 100 | `hunt_verdicts_domain.go:56` | `hunt_verdicts_domain_test.go:44`（130→100、−5→0） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | **數值範圍：** 槓桿 1–5（做多 / 做空），觀望 / 避開為 0（皆夾回邊界） | 做多/做空槓桿夾 1–5；觀望/避開 0 | `hunt_verdicts_domain.go:47-48,88`；`dependencies.go:128-129` | `hunt_verdicts_domain_test.go:44`（20→5、0→1）、`:88`（watch/avoid 為 0）；`dependencies_test.go:23` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | **數值範圍：** 部位 0%–10%，觀望 / 避開為 0（皆夾回邊界） | 部位夾 0–10%；觀望/避開 0 | `hunt_verdicts_domain.go:48,89`；`dependencies.go:130` | `hunt_verdicts_domain_test.go:44`（30→10%、−2→0）、`:88` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | **數值範圍：** 停損距離 1%–50%（皆夾回邊界） | 停損距離 <1% → 1%；>50% → 50% | `hunt_verdicts_domain.go:78`；`dependencies.go:131-132` | `hunt_verdicts_domain_test.go:63`（0.5→1%）、`:44`（60→50%） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | **數值範圍：** 停利距離 1%–200%（皆夾回邊界） | 停利距離 <1% → 1%；>200% → 200% | `hunt_verdicts_domain.go:79`；`dependencies.go:133-134` | `hunt_verdicts_domain_test.go:44`（只測 500→200%）；`dependencies_test.go:23`（只檢查政策常數） | **shallow**：下限（<1% 夾到 1%）無任何行為測試；若 `:79` 的 `decimal.Max(Minimum…)` 被拿掉，測試仍全綠 | produces-oracle | 🟠 mis-asserted |
| BR-7 | **停損停利價：** 以裁決當下的最新價格換算：做多停損價 = 價 ×（1 − 停損距離）、停利價 = 價 ×（1 + 停利距離）；做空相反。金額以精確小數計算。觀望 / 避開不設停損停利。 | 做多 P(1−d)/P(1+d)；做空 P(1+d)/P(1−d)；精確小數；觀望/避開無停損停利 | `hunt_verdicts_domain.go:78-94`（`shopspring/decimal`） | `hunt_verdicts_domain_test.go:63`（字串精確）、`:88`（watch/avoid 停損停利為空）；`hunt_verdict_application_test.go:210`（裁決當下 Bybit 0.2 → 0.18） | asserts-oracle | produces-oracle | ✅ conforms（另見「規格觀察」S-1） |
| BR-8 | **查不到最新價格：** 做多 / 做空改判觀望（槓桿、部位歸 0），理由「查不到最新價格，無法設定停損」。 | 做多/做空無價 → 觀望、槓桿 0、部位 0、理由固定字串 | `hunt_verdicts_domain.go:68-76` | `hunt_verdicts_domain_test.go:88`（long 無市場、long 價 0、short 有市場無價） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-9 | **對應：** 洞察中每一枚分析成功的幣恰好一筆裁決；CIO 漏的 → 觀望「CIO 未給出裁決」；多給的 → 忽略。代號比對不分大小寫。 | 每枚分析成功的幣恰一筆；漏 → 觀望＋理由；多 → 忽略；大小寫不敏感 | `hunt_verdicts_domain.go:35-98`；`hunt_verdict_service.go:81-89` | `hunt_verdicts_domain_test.go:122`（" pengu " 對上 PENGU、doge 忽略、STRK 補觀望）；`hunt_verdict_application_test.go:136`（失敗洞察 FAILED 不進素材） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-10 | **重問：** CIO 回覆不可用（格式不合、拒答、截斷）重問一次；仍不可用 → 輪次失敗「AI 回覆格式不合格」；AI 服務出錯不重問，輪次失敗並寫明原因。每次詢問恰好一次呼叫，伺服器端拒答備援同洞察切片。 | 不可用 → 再問恰一次；仍不可用 → 失敗＋固定原因；服務錯誤 → 不重問、失敗＋原因；每次詢問一次 HTTP 呼叫；啟用伺服器端拒答備援 | `hunt_verdict_service.go:93-114`；`claude_analysis_proxy.go:42,78-108,129-139`；`claude_verdict_wire.go:52-78` | `hunt_verdict_application_test.go:169,198`；`claude_verdict_test.go:58`（fallbacks=default、beta header、1 次請求）、`:92`（refusal/max_tokens/非 JSON/缺欄位 → 不可用；503 → 非不可用；皆 1 次請求） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-11 | **改寫結果表：** 先保存本輪裁決紀錄（輪次歷史），再以**一筆交易**改寫結果表：本輪有的幣覆蓋（無則新增）並以本輪時間為計算時間、本輪沒有的幣刪除。改寫失敗 → 輪次失敗、結果表不變（已保存的裁決紀錄留作該失敗輪次的歷史）。 | 先存紀錄再改寫；單一交易內覆蓋/新增（計算時間＝本輪）並刪缺席者；改寫失敗 → 輪次失敗、表不變、紀錄保留 | `hunt_verdict_service.go:116-135`；`hunt_board_repository.go:21-51` | `hunt_verdict_application_test.go:240`（存紀錄失敗時未呼叫 Rewrite；改寫失敗 → 輪次失敗）；`verdict_repositories_test.go:31,77,86`（覆蓋保留 ID、刪除、失敗回滾表不變） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-12 | **AI 設定（可調整）：** 模型預設 Claude Opus 5.5、思考深度預設「高」（綜合判斷）；單次詢問逾時 180 秒 | 預設模型 Opus 5.5、思考深度高、可由設定覆寫；單次詢問 180 秒逾時 | `application_config.go:115,160-165`；`claude_analysis_proxy.go:116-127`；`dependencies.go:153` | `application_config_test.go:62`（預設與覆寫）；`claude_verdict_test.go:58`（model/effort 送出）、`:128`（逾時放棄） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-13 | **AI 設定：** 市場結構每來源逾時 15 秒，查不到即無最新價格。 | 每個交易所來源 15 秒逾時；全查不到 → 無最新價格 | `perpetual_market_structure_service.go:28-41`；`application_config.go:164`；`dependencies.go:185-186` | `application_config_test.go:62`（15s）；`coin_insight_application_test.go:475`（共用 service 的每來源截斷）；`hunt_verdict_application_test.go:210`（失敗略過、全無 → nil → 觀望） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-14 | **輪次：** 重啟殘留執行中 → 失敗（被重啟中斷）。 | 啟動時所有執行中輪次（含裁決）→ 失敗，原因「被重啟中斷」 | `pipeline_run_service.go:39-54`；`pipeline_run_repository.go:87-95`（不分步驟）；`main.go:39` | `pipeline_run_application_test.go:31` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-15 | **Flow：** 找最新成功洞察（無 → 拒絕）→ 取其分析成功的洞察 → 建裁決輪次（串上洞察輪次）→ 為每枚幣重新取市場結構 → 整輪問 CIO 一次（不可用重問一次）→ 正規化每枚幣的裁決 → 保存本輪裁決紀錄 → 以一筆交易改寫獵捕結果表 → 輪次成功。 | 依序完成各步；只用分析成功的洞察；裁決當下重取市場結構；整輪一次詢問含全部幣 | `hunt_verdict_service.go:51-143` | `hunt_verdict_application_test.go:136`（素材恰 2 枚、含當下市場結構、失敗洞察排除、成功）、`:210` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-16 | **Edge Case：** 查看某一輪裁決時輪次編號不是正整數 → 「輪次編號必須是正整數」 | 非正整數（含非數字、0、負數）→ 被告知「輪次編號必須是正整數」 | `utilities/pipeline_run_id.go`（`PipelineRunIDFrom`）；`hunt_verdict_controller.go:47-51` | `hunt_verdict_controller_test.go:129`（abc）；共用 helper 的 0 案例見 `coin_discovery_controller_test.go:73` | asserts-oracle | produces-oracle | ✅ conforms（裁決路由本身只測非數字） |
| BR-17 | **Edge Case：** 存在但非裁決步驟的輪次 → 空清單。 | 查一個存在但不是裁決步驟的輪次 → 回空清單 | `hunt_verdict_service.go:163-176` | `hunt_verdict_controller_test.go:138`「a known run」 | **shallow**：mock 輪次未標步驟、mock repository 直接回 `[]`，且只斷言 HTTP 200、未斷言回應為空清單；沒有任何測試建立「非裁決步驟」的輪次 | produces-oracle | 🟠 mis-asserted |
| NFR-1 | **Cost：** 每輪 CIO 至多 2 次呼叫；市場結構資料免費。 | 每輪對 CIO 的實際呼叫 ≤ 2 | `hunt_verdict_service.go:94-97`；`claude_analysis_proxy.go:42`（`WithMaxRetries(0)`） | `hunt_verdict_application_test.go:169,198`（Times 精確）；`claude_verdict_test.go:92`（每次詢問 1 次請求） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-2 | **Consistency：** 獵捕結果表不會出現「一半新一半舊」的狀態。 | 改寫全有或全無 | `hunt_board_repository.go:22-45`（單一 Transaction） | `verdict_repositories_test.go:86`（刪除後插入失敗 → 整體回滾） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-3 | **Security：** AI 金鑰只從環境變數讀取，不出現在任何紀錄中。 | 金鑰僅來自環境變數；錯誤訊息/失敗原因/日誌不含金鑰 | `application_config.go:147`；`claude_analysis_proxy.go:42`；`main.go` 日誌未含 config | `application_config_test.go`（ANTHROPIC_API_KEY 讀取）；`claude_verdict_test.go:92`（錯誤不含 "sk-ant-secret"，失敗原因即此錯誤） | asserts-oracle | produces-oracle | ✅ conforms |

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| `internal/domain/models/domains/hunt_verdicts_domain.go:39-41` | CIO 對同一枚幣（不分大小寫）給出多筆裁決時，**只採第一筆**，其餘靜默丟棄（測試 `hunt_verdicts_domain_test.go:122` 有釘住）。PRD 只規定「漏給」「多給（洞察以外的幣）」，未規定重複 | undocumented |
| `internal/domain/models/domains/hunt_verdicts_domain.go:72` | 最新價格為 0 或負數視同「查不到最新價格」→ 改判觀望 | undocumented（與 BR-8 精神一致，建議補進 PRD） |
| `internal/infrastructure/analysis/claude_verdict_wire.go:70-71` | 信心與槓桿若 CIO 給出小數，**先四捨五入**再夾值（例：4.6 → 5） | undocumented |
| `internal/infrastructure/analysis/claude_verdict_wire.go:53-61,101-102` | 任一枚幣的裁決缺欄位即判整份回覆「不可用」（觸發重問／整輪失敗），而非只把該幣記為觀望 | undocumented（落在 BR-10「格式不合」的解讀範圍內，但粒度未定義） |
| `internal/infrastructure/persistence/hunt_board_repository.go:56` | 同信心時依幣種代號字母序排列 | undocumented |
| `internal/domain/service/hunt_verdict_service.go:128-134` + `hunt_verdict_controller.go:28-31` | 儲存紀錄或改寫結果表失敗時，輪次標為失敗，**且**手動觸發得到伺服器錯誤（而非回傳失敗輪次）；與 AI 失敗時回傳失敗輪次的行為不對稱 | undocumented（PRD 只定義輪次失敗，未定義交易者看到什麼） |
| `internal/domain/models/entities/coin_verdict.go:19` / `hunt_board_entry.go:20` | 每筆裁決另記「參考價格」（換算停損停利所用的最新價格） | undocumented（UL-MAP「裁決」未列此欄） |

Out-of-Scope 違規：**無**。

## 規格觀察（非計分）

- **S-1（規格本身的風險）：** BR-7 + BR-6 允許停利距離最高 200%，做空停利價 = 價 ×（1 − 停利距離）。停利距離 ≥ 100% 時，做空停利價會 **≤ 0**（例：0.01 × (1 − 1.5) = −0.005）。程式 `hunt_verdicts_domain.go:84` 忠實照 PRD 換算，因此不算違規，但 PRD 需要決定做空停利距離的上限（例如 < 100%）或價格下限。
- **S-2（組裝根未測）：** 裁決步驟的 AI 設定與市場結構逾時在 `cmd/server/dependencies.go:153,185-186` 組裝，沒有測試釘住「裁決用的是 `Verdict.*` 設定」；若誤接成 `Insight.*`（effort `low`），所有測試仍會通過。目前讀碼確認接線正確。

## Summary

- Conforms: 35/37 clauses ✅ (94.6%)
- Violations: —
- Mis-asserted: BR-6, BR-17
- Partial: —
- Gaps: —
- Unclear: —
- Orphans: 7（皆 undocumented，無 out-of-scope 違規）

---

## Resolution（稽核後處理，2026-10-05）

| Clause / 項目 | 原狀態 | 處理 | 驗證 |
| :--- | :--- | :--- | :--- |
| BR-6 | 🟠 mis-asserted | 新增「停利距離 0.2% → 1%、停利價 0.0101」測試（commit `74ca611`） | 拿掉 1% 下限的 mutation 被抓到 |
| BR-17 | 🟠 mis-asserted | 新增「其他步驟的輪次查裁決 → 空清單」路由測試，斷言回應為 `[]` | — |
| PRD 缺陷：做空停利價可能 ≤ 0 | 備註 | PRD 改為**做空停利距離上限 90%**；domain 以 `MaximumShortTakeProfitPercent` 夾值，做空 150% → 90%、停利價 0.001；做多仍可到 200% | 拿掉做空上限的 mutation 被抓到 |
| 裁決步驟的 AI 設定接線 | 備註 | 組裝根抽出 `claudeModelSettingsFor`，測試洞察與裁決各用自己的模型 / effort / 逾時 | 把裁決 effort 接成洞察 effort 的 mutation 被抓到 |
| Orphans 1–7 | ⚠️ | 全部寫入 PRD §4：同幣多筆採第一筆、價格 ≤ 0 視為無價、信心與槓桿小數先四捨五入、單筆缺欄位即整份不合格、同信心依代號排序、AI 失敗回失敗輪次而儲存失敗回伺服器錯誤；參考價格寫入 UL-MAP | — |

處理後：**37 conforms · 0 violation · 0 mis-asserted · 0 orphan**。全套測試（含 PostgreSQL 儲存層）通過，`internal/` 非 mock 程式碼覆蓋率 100%。
