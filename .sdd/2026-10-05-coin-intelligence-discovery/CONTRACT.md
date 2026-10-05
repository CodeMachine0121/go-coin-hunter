# Contract Traceability Matrix — 新幣情報探索（coin-intelligence-discovery）

Contract: PRD.md（v1.0，於 `22ad1de` 更新 §4 規則與 Edge Cases）
Design map: ARCH.md（§7 Traceability）
Implementation: `internal/`、`cmd/server/`（branch `feature/coin-intelligence-discovery`，稽核時 HEAD = `22ad1de`）
Oracle: Acceptance Criteria + Core Business Rules + Edge Cases + NFR（42 clauses）

> **稽核天花板：** 本文件為**靜態**契約一致性稽核——依 PRD 推得的 oracle，分別判讀測試斷言與正式程式碼路徑，不撰寫新探針、不執行自創情境。對應測試僅作佐證：以 `TEST_POSTGRES_DSN`（`go_coin_hunter_test`）、`-p 1 -count=1` 執行 `./internal/...` 全綠，persistence 儲存測試**實際執行、無 skip**。判定一律以 oracle 比對為準，不以 pass/fail 為準。
>
> **Oracle 先行：** 「Spec-expected」欄於開啟任何程式碼／測試前，僅依更新後的 PRD 文字推得。
>
> **本輪重跑：** 前一輪（HEAD `e45d521`）的發現已由 `6a5080f`（程式碼＋測試）與 `22ad1de`（PRD／ARCH 對齊）處理；本表為依更新後 PRD 從頭重做的結果，非增量修改。

## Clauses

`Spec-expected` 欄為業務可觀察的 oracle；稽核欄檢查其經 UL-MAP／ARCH 橋接後的具體產物（如「找不到這個輪次」→ `ErrPipelineRunNotFound` → 404；「本輪候選幣」→ 以該輪 `PipelineRunID` 落地的 `CoinCandidate`；「連線逾時」→ `domains.InformationSourceTimedOutReason`）。

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | Scenario: 不同來源提到同一枚幣只成為一個候選幣 — Given 幣安上幣公告提到 CT / And Bybit 上幣公告也提到 CT / When 執行一輪探索 / Then 本輪候選幣只有一個 CT / And CT 記為被 2 個來源、2 則情報提到 | 本輪候選幣恰 1 個且為 CT；來源數 = 2、情報數 = 2 | `domain/models/domains/coin_candidate_selection_domain.go:36-60` | `application/tests/coin_discovery_application_test.go:146` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | Scenario: 不同幣各自成為候選幣 — Given 幣安上幣公告提到 CT / And DEX Screener 新幣看板提到 PUMP / When 執行一輪探索 / Then 本輪候選幣為 CT 與 PUMP 兩個 | 候選幣集合恰為 {CT, PUMP} | `coin_candidate_selection_domain.go:36-70` | `coin_discovery_application_test.go:156` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | Scenario: 已收過的情報不重複保存 — Given 幣安上幣公告中提到 CT 的同一則情報在上一輪已被保存 / And 該情報仍在探索時間窗內 / When 再執行一輪探索 / Then 這則情報的保存份數仍為 1 / And CT 仍為本輪候選幣 | 該則情報只存 1 份；CT 為第 2 輪候選幣 | `infrastructure/persistence/coin_intelligence_repository.go:21-33`；`domain/service/coin_discovery_service.go:97-110` | `persistence/tests/discovery_repositories_test.go:23`（實跑）；`coin_discovery_application_test.go:182`（含情報數 = 1） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | Scenario: 時間窗內的情報產生候選幣 — Given 探索時間窗為 72 小時 / And 一則提到 CT 的情報發布於本輪開始前 10 小時 / When 執行一輪探索 / Then CT 為本輪候選幣 | 10h 前 → CT 為候選幣 | `coin_candidate_selection_domain.go:22-24,39-40`；`coin_intelligence_repository.go:35-46` | `coin_discovery_application_test.go:249` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | Scenario: 剛好落在時間窗邊界的情報仍產生候選幣 — Given 探索時間窗為 72 小時 / And 一則提到 CT 的情報發布於本輪開始前恰好 72 小時 / When 執行一輪探索 / Then CT 為本輪候選幣 | 恰 72h（含邊界）→ CT 為候選幣 | 同上 | `coin_discovery_application_test.go:250`；`discovery_repositories_test.go:37`（實跑） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-6 | Scenario: 超出時間窗的情報不產生候選幣 — Given 探索時間窗為 72 小時 / And 唯一一則提到 CT 的情報發布於本輪開始前 72 小時又 1 分鐘 / When 執行一輪探索 / Then CT 不在本輪候選幣中 | 72h+1min → CT 不在候選幣 | 同上 | `coin_discovery_application_test.go:251`；`discovery_repositories_test.go:37` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | Scenario: 排除幣種不成為候選幣 — Given CoinGecko 熱門排行出現 ETH / When 執行一輪探索 / Then ETH 不在本輪候選幣中 | ETH 不在候選幣 | `coin_candidate_selection_domain.go:31-40` | `coin_discovery_application_test.go:268`（an excluded major coin） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | Scenario: 傳統金融商品不成為候選幣 — Given 幣安新上架的永續合約 NKE 是股票永續合約 / When 執行一輪探索 / Then NKE 不在本輪候選幣中 | NKE 不在候選幣 | `informationsource/binance_perpetual_contract_information_source_proxy.go`（`IsTraditionalAsset`）＋ `coin_candidate_selection_domain.go:39` | `coin_discovery_application_test.go:280`；`informationsource/tests/information_source_proxies_test.go:58` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | Scenario: 看不出幣種的情報被保存但不產生候選幣 — Given 幣安公告「系統維護通知」沒有提到任何幣種代號 / When 執行一輪探索 / Then 這則情報被保存且沒有幣種代號 / And 本輪候選幣不因這則情報增加 | 有 1 筆該情報保存且代號為空；候選幣數不增 | `domains/information_item_domain.go:37-42`；`coin_candidate_selection_domain.go:39` | `coin_discovery_application_test.go:268`（an announcement naming no coin） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | Scenario: 全部來源成功且有候選幣 — Given 6 個資訊來源都成功 / And 情報中有 3 枚符合條件的幣 / When 執行一輪探索 / Then 輪次狀態為「成功」 / And 本輪候選幣為 3 個 / And 6 個來源結果皆為成功 | 成功；3 個候選幣；6 個來源結果皆成功 | `coin_discovery_service.go:61-134`；`domains/pipeline_run_domain.go` `ConcludeDiscovery` | `coin_discovery_application_test.go:337` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-11 | Scenario: 部分來源失敗仍算成功 — Given OKX 上幣公告無法取得，原因為「連線逾時」 / And 其餘 5 個來源成功，情報中有 2 枚符合條件的幣 / When 執行一輪探索 / Then 輪次狀態為「成功」 / And OKX 的來源結果為失敗，原因為「連線逾時」 / And 本輪候選幣為 2 個 | 成功；OKX 失敗且原因「連線逾時」；2 個候選幣 | `coin_discovery_service.go:74-81`；`information_source_results_domain.go:30-45` | `coin_discovery_application_test.go:341`；`:211`（真實逾時 → 「連線逾時」） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-12 | Scenario: 來源都成功但沒有任何候選幣 — Given 6 個資訊來源都成功 / And 所有情報都屬排除幣種或沒有幣種代號 / When 執行一輪探索 / Then 輪次狀態為「無資料」 / And 本輪候選幣為 0 個 | 無資料；0 個候選幣 | `pipeline_run_domain.go` `ConcludeDiscovery` | `coin_discovery_application_test.go:345` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-13 | Scenario: 全部來源失敗 — Given 6 個資訊來源都無法取得 / When 執行一輪探索 / Then 輪次狀態為「失敗」 / And 失敗原因為「所有資訊來源皆失敗」 / And 本輪沒有候選幣 | 失敗；原因「所有資訊來源皆失敗」；本輪無任何候選幣 | `coin_discovery_service.go:102-104`；`pipeline_run_domain.go` | `coin_discovery_application_test.go:349`；`:198`（窗內有舊情報仍 0 候選） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-14 | Scenario: 服務重啟中斷執行中的輪次 — Given 有一輪探索的狀態為「執行中」 / When 服務重新啟動 / Then 該輪狀態變為「失敗」 / And 失敗原因為「被重啟中斷」 | 失敗；原因「被重啟中斷」 | `cmd/server/main.go:39`；`domain/service/pipeline_run_service.go:39-54` | `application/tests/pipeline_run_application_test.go:31`；`discovery_repositories_test.go:101`（FindRunning，實跑） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-15 | Scenario: 手動觸發的輪次記為手動 — When 交易者手動觸發一輪探索 / Then 該輪的觸發來源為「手動」 | 觸發來源 = 手動 | `application/coin_discovery_application.go:20-24` | `coin_discovery_application_test.go:379` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-16 | Scenario: 最新候選幣只看最新一輪成功的探索 — Given 第 1 輪探索成功，候選幣為 CT 與 PUMP / And 第 2 輪探索成功，候選幣為 CT / When 交易者查看最新候選幣 / Then 看到的候選幣只有 CT | 看到 {CT} | `coin_discovery_service.go` `GetLatestCoinCandidates`；`persistence/pipeline_run_repository.go:60-74` | `discovery_repositories_test.go:60`（實跑）；`coin_discovery_application_test.go:499` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-17 | Scenario: 失敗輪次不取代上一輪成功的候選幣 — Given 第 1 輪探索成功，候選幣為 CT / And 第 2 輪探索失敗 / When 交易者查看最新候選幣 / Then 看到的候選幣只有 CT | 看到 {CT} | 同上 | `discovery_repositories_test.go:60`（a later failed run…） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-18 | Scenario: 從未成功過時最新候選幣為空 — Given 尚無任何成功的探索輪次 / When 交易者查看最新候選幣 / Then 看到空的候選幣清單 | 空清單（非錯誤） | `coin_discovery_service.go` `GetLatestCoinCandidates`（`!found` → 空切片） | `discovery_repositories_test.go:60`（never succeeded）；`controller/tests/coin_discovery_controller_test.go`（latest candidates → `[]`） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-19 | Scenario: 輪次歷史由新到舊 — Given 第 1 輪與第 2 輪探索都已結束 / When 交易者查看輪次歷史 / Then 第 2 輪排在第 1 輪之前 | 第 2 輪在第 1 輪之前 | `pipeline_run_repository.go:76-85` | `discovery_repositories_test.go:101`（實跑） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-20 | Scenario: 查看某一輪的情報 — Given 第 1 輪探索保存了 2 則情報 / When 交易者查看第 1 輪的情報 / Then 看到這 2 則情報，各自附來源、幣種代號、標題、原文連結與發布時間 | 恰 2 則，各含來源、代號、標題、連結、發布時間 | `coin_discovery_service.go` `GetCoinIntelligencesOfPipelineRun`；`entities/coin_intelligence.go:23-32` | `coin_discovery_application_test.go:526`；`discovery_repositories_test.go:37,150` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-21 | Scenario: 查看不存在的輪次 — When 交易者查看一個不存在的輪次的情報 / Then 被告知「找不到這個輪次」 | 被告知「找不到這個輪次」 | `pipeline_run_repository.go:50-52`；`domains/pipeline_run_errors.go:5`；`controller/coin_discovery_controller.go:51-53` | `coin_discovery_application_test.go:526`（unknown run）；`coin_discovery_controller_test.go:63`（404） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-1 | **每個來源每輪最多取最新 50 則**情報。 | 每來源每輪 ≤ 50 則且為最新者 | `config/application_config.go:56-57,79`；各 Proxy 截斷 | `coin_discovery_application_test.go`（以 50 呼叫 mock）；`information_source_proxies_test.go:37`（6 來源截斷，含 coingecko／dex cut at the limit）；`config/tests/application_config_test.go:48` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | **幣種代號辨識：** 合約清單以合約名稱去掉計價幣（如 CTUSDT → CT）；公告以標題中的括號代號（如「Binance Will List Cotton (CT)」→ CT、「Binance Will List Sonic (S)」→ S）或合約名稱（「CTUSDT Perpetual」→ CT）辨識；鏈上看板與熱門排行直接採用來源給的代號。代號為 1–15 個大寫英數字；**純數字（如年份「(2026)」）與常見非代號字（UTC、GMT、USD、AM、PM、FAQ、KYC、API）不算代號**。辨識不出即無幣種代號。一則公告提到多枚幣時，**每枚幣各算一則情報**。 | CTUSDT→CT；(CT)→CT；(S)→S；CTUSDT Perpetual→CT；看板／排行用來源代號；純數字與 8 個非代號字不算；辨識不出 = 無代號；多幣 = 每幣一則 | `domains/announcement_title_domain.go:9-63`；`information_item_domain.go:30-57`；perpetual proxy（`baseAsset`） | `domains/tests/announcement_title_domain_test.go:10`（含 (S)、(UTC)、(2026)）；`information_item_domain_test.go:14`；`information_source_proxies_test.go:58,144` | asserts-oracle | produces-oracle | ✅ conforms（註：8 個非代號字只測了 UTC，其餘 7 個靠同一份 map） |
| BR-3 | 代號一律大寫比對 | 代號（含排除名單）以大寫比對 | `information_item_domain.go:32`；`announcement_title_domain.go:53`；`coin_candidate_selection_domain.go:33` | `information_item_domain_test.go:14`（" pump "→PUMP）；`coin_candidate_selection_domain_test.go:13`（"eth" 排除 ETH） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | 不同鏈上同代號視為同一枚幣。 | 兩鏈同代號 → 一個候選幣 | `coin_candidate_selection_domain.go:44-56` | `coin_candidate_selection_domain_test.go:31` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | **沒有發布時間的情報**（如熱門排行）以**首次被收到的時間**作為發布時間。 | 無發布時間 → 首次收到時間，之後不變 | `information_item_domain.go:25-28`；`coin_intelligence_repository.go:28` | `information_item_domain_test.go:14`（no publish time）；`discovery_repositories_test.go:23`（首筆保留，實跑） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | **鏈上新幣看板的發布時間是該幣「最早一個交易對的建立時間」**：新幣看重的是它何時開始交易，而不是何時被看板收錄；最早交易對已超過探索時間窗的幣即使今天才上看板也不是新幣。查不到交易對時才以首次被收到的時間為準。 | DEX 情報發布時間 = 最早交易對建立時間；查不到交易對 → 首次收到時間 | `informationsource/dex_screener_token_profile_information_source_proxy.go:63-90` | `information_source_proxies_test.go:144`（取 1000 而非 2000／0；未知代幣無發布時間）；`information_item_domain_test.go:14` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-7 | **同一則情報**的判斷：同一來源 + 同一原文識別（連結或來源給的編號）+ 同一幣種代號。 | 三者皆同 → 一份；任一不同 → 各自保存 | `entities/coin_intelligence.go:13-16` | `discovery_repositories_test.go:23,150`（實跑） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-8 | 探索時間窗與排除幣種名單可由營運者調整；**每來源上限（50 則）與逾時（15 秒）是固定規則，不開放調整。** | 時間窗、排除名單可調；50 則與 15 秒不受設定影響 | `application_config.go:56-57,77-80` | `application_config_test.go:30,48` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-9 | 輪次狀態：任一來源成功且候選幣 ≥1 → 成功；任一來源成功且候選幣 = 0 → 無資料；全部來源失敗 → 失敗。 | 依三條規則決定狀態 | `pipeline_run_domain.go` `ConcludeDiscovery` | `domains/tests/pipeline_run_domain_test.go:12`；`coin_discovery_application_test.go:326` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-10 | 輪次紀錄寫不進去，該輪整體視為失敗。 | 輪次相關紀錄寫入失敗 → 該輪失敗、觸發者得到失敗 | `coin_discovery_service.go:51-59,92-126` | `coin_discovery_application_test.go:389,439,461` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-11 | Edge: 來源回應格式不符預期（含回應中缺少應有的欄位）→ 視為該來源失敗，原因寫明。 | 任何格式不符，包括缺少應有欄位 → 該來源失敗，原因寫明 | `informationsource/json_request.go`；bybit `:36-44`；okx `:36-42`；binance ann `:39-44`；perpetual／coingecko nil 檢查 | `information_source_proxies_test.go:177`（只斷言有 `Error`，未斷言原因） | shallow | diverges | 🔴 violation |
| BR-12 | Edge: 單一來源回應慢 → 每個來源各自有逾時，不拖住整輪；逾時的來源失敗原因為「連線逾時」。 | 慢來源在自己的逾時後失敗、原因「連線逾時」；其餘來源與整輪照常 | `coin_discovery_service.go:66-81`；`domains/pipeline_run_domain.go:13`；`cmd/server/dependencies.go:21-22` | `coin_discovery_application_test.go:211` | asserts-oracle | produces-oracle | ✅ conforms（註：mock 直接回傳 `ctx.Err()`；真實 HTTP 路徑靠 `getJson` 的 `%w` 包裝讓 `errors.Is` 成立，沒有 proxy 層測試證明） |
| BR-13 | Edge: 全部來源都失敗的輪次**不產生任何候選幣**，即使時間窗內留有前幾輪的情報。 | 全敗輪 0 候選幣，即使窗內有舊情報 | `coin_discovery_service.go:101-104` | `coin_discovery_application_test.go:198` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-14 | Edge: 輪次結論本身寫不進去時，該輪暫留「執行中」，於下次服務重啟時收尾為「失敗（被重啟中斷）」。 | 結論寫入失敗 → 該輪保持「執行中」（不再嘗試改寫）；重啟後變失敗（被重啟中斷） | `coin_discovery_service.go:128-131`；`pipeline_run_service.go:39-54` | `coin_discovery_application_test.go:453`（只斷言回傳錯誤）；`pipeline_run_application_test.go:31` | shallow | produces-oracle | 🟠 mis-asserted |
| BR-15 | Edge: 查看某一輪的情報時，輪次編號不是正整數 → 被告知「輪次編號必須是正整數」。 | 非正整數 → 被告知「輪次編號必須是正整數」 | `controller/coin_discovery_controller.go:43-46` | `coin_discovery_controller_test.go:70-71`（只斷言 400，未斷言訊息） | shallow | produces-oracle | 🟠 mis-asserted |
| BR-16 | Edge: 服務重啟 → 殘留的執行中輪次改為失敗（被重啟中斷）。 | 同 AC-14 | 同 AC-14 | 同 AC-14 | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-1 | Performance: 一輪探索在所有來源都正常時於 30 秒內完成 | 全部正常時一輪 ≤ 30 秒 | 來源並行、每個都受 15 秒 deadline 約束：`coin_discovery_service.go:61-85` | — | no-test | produces-oracle | 🟡 partial |
| NFR-2 | Performance: 每個來源逾時上限 15 秒。 | 每來源最多 15 秒，不可調 | `application_config.go:57,80`；`coin_discovery_service.go:66-67` | `application_config_test.go:48`；`coin_discovery_application_test.go:211`（deadline 確實會切斷） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-3 | Security: 僅本人使用，無鑑權。 | 路由不需鑑權 | `cmd/server/dependencies.go` `registerRoutes` | — | no-test | produces-oracle | 🟡 partial |
| NFR-4 | Cost: 只使用免費、無需付費方案的資訊來源。 | 只呼叫免費、無金鑰來源 | `dependencies.go` `informationSourcesFor`；6 個 Proxy 皆無 API key | — | no-test | produces-oracle | 🟡 partial |
| NFR-5 | Analytics / Tracking: 輪次紀錄即為追蹤資料。 | 輪次歷史帶狀態、原因與各來源結果 | `pipeline_run_repository.go:76-85`；`entities/pipeline_run.go:24-41` | `discovery_repositories_test.go:101`（實跑）；`coin_discovery_application_test.go:326` | asserts-oracle | produces-oracle | ✅ conforms |

### 不符合條款詳述

- **BR-11 🔴**（前一輪修了一部分，仍未全面）：回應頂層的缺欄位現在會判失敗。Bybit 的 `{}`、Binance 合約的 `{}`、CoinGecko 的 `{}` 由 nil 檢查擋下；Binance 公告與 OKX 的 `{}` 由 code 檢查擋下。但 **code 正常、清單欄位缺少**的回應仍被判為**成功、0 則**：
  - `infrastructure/informationsource/bybit_announcement_information_source_proxy.go:36-44`：`{"retCode":0,"retMsg":"OK"}`（缺 `result` 或 `result.list`）會通過。
  - `okx_announcement_information_source_proxy.go:36-42`：`{"code":"0"}`（缺 `data`）會通過。
  - `binance_announcement_information_source_proxy.go:39-44`：`{"code":"000000","data":{"catalogs":[{}]}}`（catalog 缺 `articles`）會通過。

  這三種都屬 PRD 明文「回應中缺少應有的欄位」。它們會讓壞掉的來源被算成成功，該輪可能悄悄落成「無資料」。測試 `information_source_proxies_test.go:177` 只用 `{}` 或錯誤碼，抓不到這些情況；而且只斷言有錯誤，沒有斷言「原因寫明」。
  修法：比照 CoinGecko，對 `Result.List`、`Data`、`Catalogs[0].Articles` 加 nil 檢查（JSON 的 `[]` 會解成非 nil 空切片，正常的空清單不受影響），並補上對應的 bad-answer 測試案例。
- **BR-14 🟠**：程式碼符合規則：結論 Update 失敗時，`coin_discovery_service.go:128-131` 直接回錯，不再改寫，輪次就留在「執行中」。但 `coin_discovery_application_test.go:453` 只斷言有回傳錯誤。它的 mock 讓每一次 Update 都失敗，所以即使程式改成「失敗後再寫一次失敗狀態」，或在別處把輪次改掉，這個測試也照樣通過。
  修法：斷言 Update 只被呼叫一次（或最後一次嘗試寫入的狀態是成功／無資料，而不是失敗），並搭配既有的重啟收尾測試。
- **BR-15 🟠**：`coin_discovery_controller_test.go:70-71` 只斷言 HTTP 400，沒有斷言回應內容是「輪次編號必須是正整數」。
  修法：斷言 body 的 `error` 欄位。
- **🟡 NFR-1 / NFR-3 / NFR-4**：程式碼符合，但沒有測試能夠斷言（30 秒完成、無鑑權、僅免費來源）。這三條屬結構性保證，可接受為已知限制，或補一個不經鑑權即可呼叫 `registerRoutes` 路由的組裝測試。

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| `infrastructure/informationsource/json_request.go:11-12` | 單一來源回應上限 8 MiB | undocumented（良性防護） |

前一輪的孤兒已全部收斂進契約：DEX 發布時間（O-1）→ BR-6，代號形狀與非代號字（O-2）→ BR-2，上限與逾時可調（O-3）→ BR-8 並已改為固定，非正整數輪次編號（O-4）→ BR-15。

Out of Scope 對照：沒有過濾、AI 洞察、投資裁決、獵捕結果表、推播或付費來源的程式碼。`backgroundJobsFor` 回傳空清單，所以沒有定時自動執行；也沒有清除過舊情報的程式碼。**沒有範圍外違規。**

## Summary

- Conforms: 36/42 clauses ✅ (86%)
- Violations: BR-11 (程式碼產出錯誤結果)
- Mis-asserted: BR-14, BR-15 (綠燈測試斷言過弱)
- Partial: NFR-1, NFR-3, NFR-4 (沒有斷言 oracle 的測試)
- Gaps: none
- Unclear: none
- Orphans: 1（良性）

---

## Resolution（稽核後處理，2026-10-05）

| Clause | 原狀態 | 處理 | 驗證 |
| :--- | :--- | :--- | :--- |
| BR-11 | 🔴 violation | Bybit `result.list`、OKX `data`、幣安 `articles` 缺漏即該來源失敗並寫明原因（commit `e7b8fd6`） | 新增 3 個壞回應測試＋空清單仍成功測試；移除檢查的 mutation 均被抓到 |
| BR-14 | 🟠 mis-asserted | 測試改為 Update 恰好一次、且該次寫入的是結論（無資料、無失敗原因）而非失敗狀態 | 補寫第二次失敗狀態的 mutation 被抓到 |
| BR-15 | 🟠 mis-asserted | 測試改為斷言回應訊息「輪次編號必須是正整數」與「找不到這個輪次」 | 改訊息的 mutation 被抓到 |
| BR-12 補充 | ✅（caveat） | 新增 proxy 層測試：真實 HTTP 停滯至 deadline 時錯誤可辨識為逾時 | — |
| NFR-1 / NFR-3 / NFR-4 | 🟡 partial | **接受為結構性保證**：30 秒由每來源 15 秒並行逾時保證（BR-12 測試）；無鑑權為組裝根未掛任何 middleware；免費來源由組裝根的固定來源清單保證 | 不另立測試 |
| Orphan（8 MiB 回應上限） | ⚠️ | 保留：防禦性技術限制，非業務行為 | — |

處理後：**39 conforms · 3 accepted partial · 0 violation · 0 mis-asserted**。全套測試（含 PostgreSQL 儲存層）通過，`internal/` 覆蓋率 100%。
