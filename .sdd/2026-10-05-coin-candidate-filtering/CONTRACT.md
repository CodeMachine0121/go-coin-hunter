# Contract Traceability Matrix — 候選幣過濾（coin-candidate-filtering）

Contract: PRD.md（v1.0；§4／§6 已在 `f4ad5ed` 修訂）
Design map: ARCH.md（§7 Traceability；§3.3 已在 `f4ad5ed` 修訂）
Implementation: `internal/domain/handler/`、`internal/domain/service/coin_filtering_service.go`、`internal/domain/service/coin_profile_service.go`、`internal/domain/models/domains/`、`internal/infrastructure/marketdata/`、`internal/infrastructure/persistence/coin_filter_result_repository.go`、`internal/application/coin_filtering_application.go`、`internal/controller/coin_filtering_controller.go`、`cmd/server/dependencies.go`、`internal/config/application_config.go`
Branch: `feature/coin-candidate-filtering`（HEAD `f4ad5ed`；重新審查，上一版為 `0370c53`）
Oracle: 驗收條件 + 核心業務規則 + 非功能需求（共 48 條：AC 30 · BR 15 · NFR 3）
審查日期：2026-10-05

> 這是**從頭重做**的靜態一致性審查。每條條文的預期結果（oracle）只從目前的 PRD 文字推導，且在打開程式碼之前就先記下（§3 沒有變動；§4／§6 的修訂條文都重新推導過）；之後才分別判斷「測試有沒有斷言這個 oracle」與「程式碼有沒有產出這個 oracle」。審查沒有自行撰寫或執行新的探測情境；只跑過已對應的既有測試（handler、domains、application、marketdata、controller、config，以及三支 filtering storage 測試，`-p 1 -count=1`），全部通過，但通過與否不作為判定依據。

## Out of Scope（反向檢查清單）

AI 洞察、投資裁決、獵捕結果表、定時自動執行、付費資料來源、以價格走勢判斷底部。→ 程式碼中**沒有**任何一項（沒有新增 job、沒有 AI 呼叫、所有來源都是免費端點）。

## Clauses

`Spec-expected` 欄是 Phase 2 從 PRD 推導出的業務可觀察結果；稽核欄則是透過 UL-MAP／ARCH 對應到具體產物（例如「淘汰」→ `rejected`、「無資料」→ `noData`、「保留」→ `IsKept`）後的判定。

### US-01 — 依規則逐枚過濾候選幣

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | Scenario: 六條規則全部通過即保留 — Given ZORA 六條過濾規則的結果皆為通過 / When 執行一輪過濾 / Then ZORA 被保留 | ZORA 被保留 | `coin_filter_verdicts_domain.go:18-26`；`coin_filtering_service.go:81-91` | `coin_filtering_application_test.go:176-210`（:197） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | Scenario: 任一規則淘汰即淘汰並寫明理由 — Given PUMP 的完全稀釋估值為 52 億美元 / And PUMP 其餘五條規則皆為通過 / When 執行一輪過濾 / Then PUMP 被淘汰 / And 完全稀釋估值規則的理由為「完全稀釋估值 52 億美元高於上限 10 億美元」 | PUMP 被淘汰，理由一字不差為「完全稀釋估值 52 億美元高於上限 10 億美元」 | `fully_diluted_valuation_filter_handler.go:42-45`；`number_description_domain.go:25-32` | `coin_filtering_application_test.go:202-204`；`filter_handlers_test.go:89-90` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | Scenario: 無資料不淘汰 — Given GRASS 查不到解鎖時程 / And GRASS 其餘五條規則皆為通過 / When 執行一輪過濾 / Then GRASS 被保留 / And 解鎖時程規則的結果為無資料 | GRASS 被保留，解鎖時程結果為無資料 | `unlock_schedule_filter_handler.go:34-37`；`coin_filter_verdicts_domain.go:18-26` | `coin_filtering_application_test.go:206-209` | asserts-oracle | produces-oracle | ✅ conforms |

### US-02 — 安全檢查

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-4 | Scenario: 沒有合約風險即通過 — Given DOUU 有合約位址 / And DOUU 不是蜜罐、買賣稅皆為 0%、不可增發、不可凍結持有人資產 / When 執行一輪過濾 / Then DOUU 的安全檢查結果為通過 | 安全檢查通過 | `security_check_filter_handler.go:39-67` | `filter_handlers_test.go:47-48` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | Scenario: 蜜罐被淘汰 — Given SCAM 是蜜罐 / When 執行一輪過濾 / Then SCAM 的安全檢查結果為淘汰 / And 理由為「蜜罐，無法賣出」 | 淘汰，理由「蜜罐，無法賣出」 | `security_check_filter_handler.go:42-47` | `filter_handlers_test.go:49-50` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-6 | Scenario: 稅率剛好 10% 仍通過 — Given TAXY 的賣出稅為 10% / When 執行一輪過濾 / Then TAXY 的安全檢查結果為通過 | 賣出稅 10% 時通過 | `security_check_filter_handler.go:51` | `filter_handlers_test.go:51-54` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | Scenario: 稅率超過 10% 被淘汰 — Given TAXZ 的賣出稅為 10.5% / When 執行一輪過濾 / Then TAXZ 的安全檢查結果為淘汰 / And 理由為「賣出稅 10.5% 超過上限 10%」 | 淘汰，理由「賣出稅 10.5% 超過上限 10%」 | `security_check_filter_handler.go:51-53` | `filter_handlers_test.go:55-56` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | Scenario: 沒有合約位址為無資料 — Given SUI 是公鏈原生幣，沒有合約位址 / When 執行一輪過濾 / Then SUI 的安全檢查結果為無資料 | 安全檢查為無資料 | `security_check_filter_handler.go:27-30`；`coin_profile_service.go:149-154` | `filter_handlers_test.go:60`；`coin_filtering_application_test.go:253` | asserts-oracle | produces-oracle | ✅ conforms |

### US-03 — 市場面門檻

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-9 | Scenario: 成交額剛好達到門檻 — Given 一枚幣最近 24 小時成交額為 100 萬美元 / When 執行一輪過濾 / Then 它的流動性門檻結果為通過 | 正好 100 萬美元時通過 | `liquidity_threshold_filter_handler.go:30` | `filter_handlers_test.go:70-71` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | Scenario: 成交額低於門檻 — Given 一枚幣最近 24 小時成交額為 99 萬美元 / When 執行一輪過濾 / Then 它的流動性門檻結果為淘汰 / And 理由為「24 小時成交額 99 萬美元低於門檻 100 萬美元」 | 淘汰，理由「24 小時成交額 99 萬美元低於門檻 100 萬美元」 | `liquidity_threshold_filter_handler.go:30-34` | `filter_handlers_test.go:72-73` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-11 | Scenario: 完全稀釋估值剛好在上限 — Given 一枚幣的完全稀釋估值為 10 億美元 / When 執行一輪過濾 / Then 它的完全稀釋估值結果為通過 | 正好 10 億美元時通過 | `fully_diluted_valuation_filter_handler.go:42` | `filter_handlers_test.go:83-84` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-12 | Scenario: 完全稀釋估值低於下限 — Given 一枚幣的完全稀釋估值為 900 萬美元 / When 執行一輪過濾 / Then 它的完全稀釋估值結果為淘汰 / And 理由為「完全稀釋估值 900 萬美元低於下限 1,000 萬美元」 | 淘汰，理由「完全稀釋估值 900 萬美元低於下限 1,000 萬美元」 | `fully_diluted_valuation_filter_handler.go:38-41` | `filter_handlers_test.go:87-88` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-13 | Scenario: 查不到完全稀釋估值 — Given 一枚幣查不到完全稀釋估值 / When 執行一輪過濾 / Then 它的完全稀釋估值結果為無資料 | 無資料 | `fully_diluted_valuation_filter_handler.go:30-33` | `filter_handlers_test.go:91-92` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-14 | Scenario: 流通比剛好 20% — Given 一枚幣流通量 2 億、最大供給量 10 億 / When 執行一輪過濾 / Then 它的流通比結果為通過 | 20% 時通過 | `circulating_ratio_filter_handler.go:43-49` | `filter_handlers_test.go:98-99` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-15 | Scenario: 流通比低於 20% — Given 一枚幣流通量 1.9 億、最大供給量 10 億 / When 執行一輪過濾 / Then 它的流通比結果為淘汰 / And 理由為「流通比 19% 低於門檻 20%」 | 淘汰，理由「流通比 19% 低於門檻 20%」 | `circulating_ratio_filter_handler.go:45-48` | `filter_handlers_test.go:100-101` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-16 | Scenario: 沒有最大供給量時以總供給量計 — Given 一枚幣沒有最大供給量 / And 流通量 3 億、總供給量 10 億 / When 執行一輪過濾 / Then 它的流通比結果為通過 | 以總供給量為分母（30%），通過 | `circulating_ratio_filter_handler.go:31-37` | `filter_handlers_test.go:105-106` | asserts-oracle | produces-oracle | ✅ conforms |

### US-04 — 解鎖時程

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-17 | Scenario: 近期解鎖量小 — Given 一枚幣流通量 10 億 / And 未來 14 天內累計解鎖 1,000 萬 / When 執行一輪過濾 / Then 它的解鎖時程結果為通過 | 1% → 通過 | `unlock_schedule_filter_handler.go:43-53` | `filter_handlers_test.go` 中 "a small upcoming unlock passes" | asserts-oracle | produces-oracle | ✅ conforms |
| AC-18 | Scenario: 近期解鎖量剛好 5% — Given 一枚幣流通量 10 億 / And 未來 14 天內累計解鎖 5,000 萬 / When 執行一輪過濾 / Then 它的解鎖時程結果為淘汰 / And 理由為「14 天內解鎖 5,000 萬，達流通量 5%（門檻 5%）」 | 淘汰，理由「14 天內解鎖 5,000 萬，達流通量 5%（門檻 5%）」 | `unlock_schedule_filter_handler.go:53-58` | `filter_handlers_test.go` 中 "exactly five percent is rejected"；`coin_filtering_application_test.go:276-289` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-19 | Scenario: 解鎖落在 14 天之外 — Given 一枚幣流通量 10 億 / And 唯一一次解鎖 2 億發生在第 15 天 / When 執行一輪過濾 / Then 它的解鎖時程結果為通過 | 第 15 天不計入，通過 | `unlock_schedule_filter_handler.go:46` | `filter_handlers_test.go` 中 "an unlock on day fifteen is outside" | asserts-oracle | produces-oracle | ✅ conforms |

### US-05 — 是否已上永續合約

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-20 | Scenario: 任一交易所有 USDT 永續合約即通過 — Given ZORA 只在 Bybit 有 USDT 永續合約 / When 執行一輪過濾 / Then ZORA 的永續合約結果為通過 | 通過 | `perpetual_contract_listing_filter_handler.go:20-28`；`coin_profile_service.go:118-125` | `filter_handlers_test.go` 中 "one exchange is enough"；`coin_filtering_application_test.go:199-200` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-21 | Scenario: 三家都沒有永續合約即淘汰 — Given DOUU 在幣安、Bybit、OKX 都沒有 USDT 永續合約 / When 執行一輪過濾 / Then DOUU 的永續合約結果為淘汰 / And 理由為「幣安、Bybit、OKX 皆無 USDT 永續合約」 | 淘汰，理由「幣安、Bybit、OKX 皆無 USDT 永續合約」 | `perpetual_contract_listing_filter_handler.go:21-24` | `filter_handlers_test.go` 中 "no exchange is rejected"；`coin_filtering_application_test.go:273` | asserts-oracle | produces-oracle | ✅ conforms |

### US-06 — 過濾輪次

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-22 | Scenario: 有保留即成功並串上探索輪次 — Given 最新一輪成功的探索為第 5 輪，候選幣 3 枚 / And 其中 1 枚六條規則都沒有淘汰 / When 執行一輪過濾 / Then 過濾輪次狀態為「成功」 / And 過濾輪次記下其來源為第 5 輪探索 | 輪次成功，來源記為第 5 輪探索 | `coin_filtering_service.go:62-68,111`；`pipeline_run_domain.go:44-53` | `coin_filtering_application_test.go:190-193`；`filtering_domains_test.go:54-60` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-23 | Scenario: 全部淘汰為無資料 — Given 最新一輪成功探索的 3 枚候選幣都被淘汰 / When 執行一輪過濾 / Then 過濾輪次狀態為「無資料」 | 輪次為無資料 | `pipeline_run_domain.go:47-49` | `coin_filtering_application_test.go:265-268`；`filtering_domains_test.go:58` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-24 | Scenario: 從未有成功的探索 — Given 從未有過成功的探索輪次 / When 交易者手動觸發過濾 / Then 被告知「尚無成功的探索輪次」 / And 沒有建立任何過濾輪次 | 告知「尚無成功的探索輪次」；沒有建立任何過濾輪次 | `coin_filtering_service.go:53-55`；`coin_filtering_controller.go:24-27` | `coin_filtering_application_test.go:291-302`（gomock 嚴格模式，沒有 `Create` 預期）；`coin_filtering_controller_test.go:66-73` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-25 | Scenario: 資料來源整體無法取得 — Given 永續合約清單整體無法取得，原因為「連線逾時」 / When 執行一輪過濾 / Then 過濾輪次狀態為「失敗」 / And 失敗原因寫明永續合約清單無法取得及其原因 / And 本輪沒有任何過濾結果 | 輪次失敗；原因點名永續合約清單並寫出「連線逾時」；沒有保存任何結果 | `coin_profile_service.go:72-79,111`；`coin_filtering_service.go:98-106` | `coin_filtering_application_test.go:509-534`（真的讓 context 逾時，斷言「資料來源無法取得：Bybit 永續合約清單（連線逾時）」）；`:304-345`（0 筆結果） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-26 | Scenario: 手動觸發的過濾輪次記為手動 — When 交易者手動觸發一輪過濾 / Then 該過濾輪次的觸發來源為「手動」 | 觸發來源為手動 | `coin_filtering_application.go:20-24`；`coin_filtering_service.go:64` | `coin_filtering_application_test.go:194` | asserts-oracle | produces-oracle | ✅ conforms |

### US-07 — 查看過濾結果

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-27 | Scenario: 最新保留的候選幣只看最新一輪成功的過濾 — Given 第 1 輪過濾保留 ZORA / And 第 2 輪過濾保留 GRASS / When 交易者查看最新保留的候選幣 / Then 只看到 GRASS / And 附六條規則各自的結果與理由 | 只看到最新一輪成功過濾中保留的 GRASS，並附上每條規則的結果與理由 | `coin_filtering_service.go:120-144`；`pipeline_run_repository.go:60-71` | `coin_filtering_application_test.go:425-438` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-28 | Scenario: 從未成功過濾時為空 — Given 尚無任何成功的過濾輪次 / When 交易者查看最新保留的候選幣 / Then 看到空清單 | 空清單 | `coin_filtering_service.go:128-130` | `coin_filtering_application_test.go:440-448`；`coin_filtering_controller_test.go:106-113` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-29 | Scenario: 查看某一輪過濾的全部結果 — Given 第 2 輪過濾淘汰 PUMP、保留 GRASS / When 交易者查看第 2 輪過濾的全部結果 / Then 看到 PUMP 為淘汰、GRASS 為保留 / And 每枚幣附六條規則各自的結果與理由 | PUMP 淘汰、GRASS 保留，每枚幣都附規則結果與理由 | `coin_filtering_service.go:147-164`；`coin_filter_result.go:21-35` | `coin_filtering_application_test.go:468-487`（以完整 DTO 比對，含代號與規則結果） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-30 | Scenario: 查看不存在的過濾輪次 — When 交易者查看一個不存在的輪次的過濾結果 / Then 被告知「找不到這個輪次」 | 告知「找不到這個輪次」 | `coin_filtering_service.go:150-152`；`pipeline_run_errors.go:5`；`coin_filtering_controller.go:55-58` | `coin_filtering_application_test.go:489-496`；`coin_filtering_controller_test.go:136` | asserts-oracle | produces-oracle | ✅ conforms |

### Section 4 — 核心業務規則

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| BR-1 | **規則清單：** 安全檢查、流動性門檻、完全稀釋估值、流通比、解鎖時程、是否已上永續合約；彼此獨立，結果只有通過 / 淘汰 / 無資料，淘汰與無資料都附理由。 | 每枚幣剛好得到六條規則的結果，每個結果都是三種之一；淘汰與無資料一律有非空的理由 | `dependencies.go:37-46`；`filter_verdict_vo.go:5-9`；每個 handler 的淘汰／無資料分支都有設定 `Reason` | `coin_filtering_application_test.go:209`；`filter_handlers_test.go`（每個淘汰／無資料案例都斷言了理由） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | **保留判定：** 任一規則淘汰 → 淘汰；否則保留。 | 有任何淘汰就淘汰，否則保留 | `coin_filter_verdicts_domain.go:18-26` | `filtering_domains_test.go:37-47` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | **門檻（預設，營運者可調整）：** 稅率上限 10%（含 10% 通過）；24 小時成交額下限 100 萬美元（含）；完全稀釋估值 1,000 萬～10 億美元（含兩端）；流通比下限 20%（含）；解鎖觀察期 14 天、解鎖量上限為流通量的 5%（達到即淘汰）。 | 預設值與邊界如條文所述，且每一項營運者都能調整 | `application_config.go:111-118`；各 handler 的比較 | `filter_handlers_test.go`（每個邊界）；`application_config_test.go`（`TestLoadReadsFilteringThresholds`） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | **安全檢查淘汰條件：** 蜜罐、無法賣出、買入稅或賣出稅超過上限、發行方仍可增發、發行方可凍結持有人資產（含暫停轉帳或黑名單）。「發行方仍可」指合約**仍有人掌控**：以太坊系的鏈上有非零的擁有者、有隱藏擁有者、或擁有權可被取回；已放棄擁有權的合約即使留有增發或凍結功能也不算（例如已放棄擁有權的 PEPE）。Solana 以增發權與凍結權是否仍存在判斷。只支援主流鏈（以太坊、BNB 鏈、Base、Arbitrum、Polygon、Solana）；其他鏈或沒有合約位址 → 無資料。 | 以下任一成立就淘汰：蜜罐／無法賣出／稅率超過上限／（仍有人掌控 且 可增發）／（仍有人掌控 且 可暫停轉帳或黑名單）。「仍有人掌控」= 非零擁有者 或 隱藏擁有者 或 擁有權可取回。Solana 看增發權與凍結權。只支援六條主流鏈，其他鏈或沒有位址 → 無資料 | `go_plus_token_security_proxy.go:42-45,67-71,91-98`；`security_check_filter_handler.go:27-63` | `market_data_proxies_test.go:106-168`（已放棄擁有權、有擁有者、隱藏擁有者、擁有權可取回、擁有者為空、Solana 增發權、Solana 凍結權、tron）；`filter_handlers_test.go:44-66` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | **安全資料查詢範圍：** 免費額度每次只能查一個位址，因此**只為至少上了一家永續合約的幣**查安全資料；沒有永續合約的幣安全檢查記為無資料（理由「未上永續合約，未查詢安全資料」），它本來就會被永續合約規則淘汰。 | 沒有永續合約的幣不發出安全查詢，安全檢查為無資料（理由「未上永續合約，未查詢安全資料」） | `coin_profile_service.go:158-177` | `coin_filtering_application_test.go:257-274`（gomock 嚴格模式，沒有 `FindTokenSecurity` 預期） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | **流通比計算：** 流通量 ÷ 最大供給量；沒有最大供給量（或最大供給量為 0）改用總供給量；兩者都沒有或流通量不明 → 無資料。 | 有最大供給量時一定以它為分母；沒有或為 0 才改用總供給量；都沒有或流通量不明 → 無資料 | `circulating_ratio_filter_handler.go:25-43` | `filter_handlers_test.go:96-114`（新增：同時有兩者時用最大供給量 → 15% 淘汰；最大供給量為 0 → 用總供給量） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-7 | **解鎖時程：** 只計「本輪開始之後、14 天之內（含第 14 天）」的解鎖；查不到該幣的解鎖時程 → 無資料；流通量不明 → 無資料。 | 只累計 本輪開始 < 解鎖時間 ≤ 本輪開始+14 天 的事件；時程或流通量不明 → 無資料 | `unlock_schedule_filter_handler.go:34-49`；`coin_filtering_service.go:61,74` | `filter_handlers_test.go`（`TestUnlockScheduleFilterHandler`）；`coin_filtering_application_test.go:283` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-8 | **永續合約：** 只算以 USDT 計價、目前可交易的永續合約；股票等傳統金融合約不算。 | 只算 USDT 計價、可交易、加密原生的永續合約 | `binance_perpetual_contract_listing_proxy.go:39`；`bybit_perpetual_contract_listing_proxy.go:44-45`；`okx_perpetual_contract_listing_proxy.go:42-43` | `market_data_proxies_test.go:183-220` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-9 | **幣的識別：** 探索時從鏈上新幣看板得知鏈與合約位址的幣，**只**以該位址比對市值資料與安全資料（市值資料庫不認得該位址時改用鏈上交易對資料，絕不以同代號的別枚幣代替；該鏈不受支援則安全檢查為無資料）；其餘以代號比對市值資料庫，同代號取市值最大的那一枚，並以它登記的合約位址做安全檢查（主流鏈優先序：以太坊、BNB 鏈、Base、Arbitrum、Polygon、Solana）。 | 有宣告位址的幣只以該位址比對；市值資料庫不認得 → 改用鏈上交易對資料，絕不改用同代號的別枚幣；宣告的鏈不受支援 → 安全檢查為無資料。其他幣以代號比對，同代號取市值最大者，依 ETH→BSC→Base→Arbitrum→Polygon→Solana 選位址 | `coin_gecko_coin_market_data_proxy.go:66-74`；`coin_profile_service.go:126-154` | `market_data_proxies_test.go:42-81`（PENGU2 不以同代號替代）；`coin_filtering_application_test.go:212-255`（DOUU 走鏈上交易對備援）、`:536-569`（tron → 無資料、ETH 優先於 BSC） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-10 | **市值資料來源優先序：** 幣種市值資料庫有該幣就用它；沒有時改用鏈上交易對資料（估值取流動性最大的交易對，24 小時成交額為該幣所有交易對加總；沒有供給量）。 | 市值資料庫優先；否則用鏈上交易對：估值取流動性最大的交易對，成交額為所有交易對加總，沒有供給量 | `coin_profile_service.go:81-95`；`dex_screener_coin_market_data_proxy.go:75-89` | `coin_filtering_application_test.go:231-252`；`market_data_proxies_test.go:83-104`（估值 3,100,000 取自流動性最大的交易對、成交額 200+300=500） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-11 | **來源整體故障：** 市值資料、鏈上安全資料、永續合約清單（三家任一家）、解鎖時程任一來源整體無法取得 → 本輪過濾失敗並寫明是哪個來源與原因（逾時寫「連線逾時」），不保存任何結果。單一幣查不到（含解鎖資料集中某一個項目讀不到）不算整體故障，該幣該項為無資料。 | 任一來源（含三家交易所任一家）**整體**故障 → 輪次失敗，寫明來源與原因（逾時寫「連線逾時」），0 筆結果；只有單一項目讀不到時 → 只有那枚幣那一項為無資料 | `coin_profile_service.go:72-79,88,111,172,181-185`；`defi_llama_token_unlock_schedule_proxy.go:54-59` | `coin_filtering_application_test.go:304-345,509-534`；`market_data_proxies_test.go:350-363` | asserts-oracle（但沒有測試涵蓋「解鎖來源在逐項讀取途中整體逾時」） | **diverges** — `defi_llama_token_unlock_schedule_proxy.go:57-59` 只要逐項讀取出錯就 `continue`，連 context 已逾時或被取消時也一樣。整個解鎖查詢共用一個 20 秒 deadline（`coin_profile_service.go:180-181`）：只要清單讀得到、但逐項讀取途中撞到 deadline，剩下每一項都會立刻失敗然後被吞掉。結果本輪以**成功**結束，那些幣的解鎖時程都記為無資料，而不是以「連線逾時」失敗。整個來源故障被誤當成多個單幣缺口（每一項都回 429 時也一樣） | 🔴 violation |
| BR-12 | **輪次：** 過濾輪次串上它所過濾的探索輪次；保留 ≥1 → 成功；保留 0 → 無資料；來源整體故障 → 失敗；輪次寫不進去即整體失敗；重啟時殘留執行中 → 失敗（被重啟中斷）。 | 串上游；≥1 → 成功；0 → 無資料；來源故障 → 失敗；寫入失敗 → 整個呼叫失敗；重啟殘留 → 失敗（被重啟中斷） | `coin_filtering_service.go:62-71,98-116`；`pipeline_run_domain.go:44-61`；`pipeline_run_service.go:38-53` | `coin_filtering_application_test.go:192,268,339,347-422`；`filtering_domains_test.go:54-60`；`pipeline_run_application_test.go:33-40` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-13 | （Edge）同一枚候選幣在一輪中只有一組結果。 | 同一輪每枚幣剛好一筆結果 | `coin_filter_result.go:8-9`；`coin_profile_service.go`（每個候選幣一份檔案） | `filtering_repositories_test.go:49-56` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-14 | （Edge）金額數值以美元精確小數保存與比較，不因浮點誤差誤判邊界。 | 每個金額都以精確小數比較 | `json_decimal.go:10-34`；所有 VO 與 handler 都用 `decimal.Decimal` | `market_data_proxies_test.go`（`"172066967.5"` 原樣保留）；`filter_handlers_test.go` 的邊界案例 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-15 | （Edge）查看某一輪的過濾結果時，輪次編號不是正整數 → 被告知「輪次編號必須是正整數」。 | 告知「輪次編號必須是正整數」 | `coin_filtering_controller.go:47-51` | `coin_filtering_controller_test.go:133` | asserts-oracle | produces-oracle | ✅ conforms |

### Section 6 — 非功能需求

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| NFR-1 | **Performance:** 一輪過濾在「60 秒 ＋ 每枚需查安全資料的幣 2 秒」內完成。安全資料的免費額度約每分鐘 30 次、每次一個合約，查詢必須間隔 2 秒，這是免費資料的物理上限；只有已上永續合約、且有主流鏈合約位址的幣才需要查。 | 一輪在 60 秒 + 2 秒 ×（已上永續合約且有主流鏈位址的幣數）內結束 | `application_config.go:78-80`（每個來源逾時 20 秒、安全查詢間隔 2 秒）；`go_plus_token_security_proxy.go:111-122`；`coin_profile_service.go:81-185` | — | no-test | **unclear** — 安全查詢的間隔是從前一次查詢**開始**算起，所以只要每次查詢少於 2 秒，就落在每枚 2 秒的額度內。但有兩點會讓上限可能被突破，光讀程式碼無法排除：(a) 非安全來源是依序呼叫的，每個都可以花到最多 20 秒才成功：CoinGecko → DEX Screener → 交易所清單（三家並行）→ 解鎖時程，最壞 80 秒，超過 60 秒的基本預算；(b) 每次安全查詢本身最多可花 20 秒，不只 2 秒。整輪沒有總 deadline | ❔ unclear |
| NFR-2 | **Cost:** 只使用免費、無需付費方案的資料來源；每輪對每個來源的查詢次數盡量批次化以符合免費頻率限制。 | 只用免費來源；查詢有批次化或節流 | `coin_gecko_coin_market_data_proxy.go:79-90`（每批 ≤250）；`dex_screener_coin_market_data_proxy.go:54-57`（每批 ≤30）；GoPlus 節流 | `market_data_proxies_test.go:170-181,378-431`（250+1、30+1、間隔） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-3 | **Security:** 僅本人使用，無鑑權。 | 端點不需鑑權 | `dependencies.go:119-122` | `coin_filtering_controller_test.go`（都以未鑑權的請求呼叫並成功） | asserts-oracle | produces-oracle | ✅ conforms |

## Orphans（沒有任何條文對應的程式碼）

| Code | Description | Verdict |
|------|-------------|---------|
| — | 上一輪的三項孤兒（輪次編號 400、鏈上交易對成交額加總、最大供給量為 0 改用總供給量）現在分別由 BR-15、BR-10、BR-6 涵蓋。沒有發現新的孤兒行為。 | — |

沒有任何程式碼落在 Out of Scope 範圍。

## Summary

- Conforms: 46/48 條 ✅（95.8%）
- Violations: BR-11（解鎖來源在逐項讀取途中逾時或被限流時，被當成單幣缺口吞掉；輪次以成功結束，而不是「連線逾時」失敗）
- Mis-asserted: —
- Partial: —
- Gaps: —
- Unclear: NFR-1（受外部延遲影響；依序的來源逾時預算最壞 80 秒，超過 60 秒基本預算，且整輪沒有總 deadline）
- Orphans: 0

---

## Resolution（第二輪稽核後處理，2026-10-05）

| Clause | 原狀態 | 處理 | 驗證 |
| :--- | :--- | :--- | :--- |
| BR-11 | 🔴 violation | 解鎖資料集只有「不存在（404）」才視為該幣無資料；逾時、取消、被拒（如 429）等一律視為整個來源無法取得 → 本輪失敗、逾時寫「連線逾時」（commit `a20e871`） | 新增「資料集被拒」「資料集之間逾時」兩個測試；把 404 判斷改成一律略過、或把錯誤改回略過的 mutation 均被抓到 |
| NFR-1 | ❔ unclear | 取資料加上整輪時限：基本預算 60 秒，每次安全查詢再加 2 秒；所有來源呼叫都在此時限內，超過即本輪失敗「連線逾時」 | 新增時限測試（基本 100ms＋每次 50ms，兩次各 150ms 的查詢 → 300ms 內以「連線逾時」失敗）；拿掉時限或讓安全查詢脫離時限的 mutation 均被抓到 |

處理後：**48 conforms · 0 violation · 0 mis-asserted · 0 partial · 0 unclear**。全套測試（含 PostgreSQL 儲存層）通過，`internal/` 非 mock 程式碼覆蓋率 100%。
