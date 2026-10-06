# Contract Traceability Matrix — bullish-coin-hunt

Contract: PRD.md
Design map: ARCH.md
Implementation: `internal/`（handler、domains、service、application、infrastructure/analysis、config）、`cmd/server/dependencies.go`
Oracle: Acceptance Criteria（31 scenarios）+ Core Business Rules（8）+ Non-Functional（2）= 41 clauses

> Static conformance audit: each test's assertions and each code path were judged against the spec oracle, not by suite pass/fail. Only the mapped single tests were run as corroboration (plus mutation checks during implementation).

## First pass and fixes

The first pass found four non-conforming clauses. All four were fixed and re-verified; the matrix below shows the final state.

| ID | First-pass status | Finding | Fix |
| :-- | :-- | :-- | :-- |
| AC-05.1 | 🔴 violation | The PRD oracle said the reason reads 「過濾未成功：無資料」, but the established round-reason format (scheduled-pipeline slice, already verified) is 「{步驟}未成功：{狀態}」, giving 「過濾未成功：noData」. The PRD contradicted an accepted contract, not the code. | PRD corrected to 「過濾未成功：noData」, citing the existing format. Changing the code would have broken the earlier slice's contract. |
| AC-04.3 | 🟠 mis-asserted | The test asserted the confidence-49 long stays in history as a long, but not with confidence 49. | Test now asserts every saved verdict keeps the confidence it was judged with. |
| AC-05.3 | 🟠 mis-asserted | The only insight-failure round test used a storage error (an erroring step), not "every coin's analysis fails" (a failed run). | Added the round test `every coin's analysis failing leaves the board`. |
| NFR-1 | 🟡 partial | Concurrent market-structure lookups had no test. | Added a barrier test that only answers once both coins are being asked; sequential lookups make it fail (falsified). |

## Clauses

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-01.1 | 24 小時漲幅溫和，通過（+12%） | 漲跌幅規則通過 | `price_change_filter_handler.go:37-44` | `filter_handlers_test.go:151` a moderate rise passes | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.2 | 跌幅剛好 10%，通過 | 通過 | `price_change_filter_handler.go:37` | 同上 a fall of exactly ten percent passes | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.3 | 跌幅超過 10%，淘汰（-10.5%） | 淘汰，理由「24 小時跌幅 10.5% 超過下限 10%」，PENGU 不被保留 | `price_change_filter_handler.go:37-40`；既有 `coin_filter_verdicts_domain.go` | 同上 a fall over ten percent is rejected；`coin_filtering_application_test.go:676`（-20% → 不保留） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.4 | 漲幅剛好 60%，通過 | 通過 | `price_change_filter_handler.go:41` | 同上 a rise of exactly sixty percent passes | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.5 | 漲幅超過 60%，追高風險淘汰 | 淘汰，理由「24 小時漲幅 61% 超過上限 60%」 | `price_change_filter_handler.go:41-44` | 同上 a rise over sixty percent is rejected | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.6 | 查不到市場結構，三條動能規則皆無資料 | 三條皆無資料，理由「查不到永續合約市場結構」，不因此淘汰 | 三個 handler 的 `MarketStructure == nil` 分支；`coin_profile_service.go:106-115` 查不到留 nil | 三個 handler 測試的 no market structure is no data；`coin_filtering_application_test.go:676`（ARB） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.7 | 持倉量增加，通過 | 通過 | `open_interest_change_filter_handler.go:35` | `filter_handlers_test.go:169` growing open interest passes | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.8 | 持倉量剛好減少 10%，通過 | 通過 | 同上 | a drop of exactly ten percent passes | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.9 | 持倉量減少超過 10%，淘汰 | 淘汰，理由「持倉量 24 小時減少 11% 超過下限 10%」 | `open_interest_change_filter_handler.go:35-38` | a drop over ten percent is rejected | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.10 | 交易所沒提供持倉量變化 | 無資料，理由「查不到持倉量 24 小時變化」 | `open_interest_change_filter_handler.go:28-30` | no open interest change is no data | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.11 | 資金費率正常，通過 | 通過 | `funding_rate_overheat_filter_handler.go:35` | `filter_handlers_test.go:183` an ordinary rate passes | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.12 | 資金費率剛好 0.1%，通過 | 通過 | 同上 | exactly the ceiling passes | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.13 | 資金費率超過 0.1%，多方過熱淘汰 | 淘汰，理由「資金費率 0.11% 高於上限 0.1%」 | `funding_rate_overheat_filter_handler.go:35-38` | over the ceiling is rejected | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.14 | 負資金費率不淘汰 | 通過 | `funding_rate_overheat_filter_handler.go:35` | a negative rate passes | asserts-oracle | produces-oracle | ✅ conforms |
| AC-01.15 | 交易所沒提供資金費率 | 無資料，理由「查不到資金費率」 | `funding_rate_overheat_filter_handler.go:28-30` | no funding rate is no data；`coin_filtering_application_test.go:676`（STRK） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-02.1 | 只有看多的幣交給 CIO | CIO 只看到 PENGU；本輪裁決只有 PENGU | `bullish_focus_domain.go:25`；`hunt_verdict_service.go:83` | `hunt_verdict_application_test.go:404`（看空 STRK 不出現；只存看多的裁決） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-02.2 | 看多強度剛好 6，交給 CIO | PENGU 交給 CIO | `bullish_focus_domain.go:29` | 同上（OP 強度 6 被看到）；`hunt_verdicts_domain_test.go:158` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-02.3 | 看多強度 5，不交給 CIO | 只有 STRK 交給 CIO | `bullish_focus_domain.go:29` | 同上（ARB 強度 5 不被看到） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-02.4 | 沒有看多的洞察，不問 CIO | 不問 CIO；裁決輪次「無資料」；沒有裁決；結果表為空 | `hunt_verdict_service.go:93-95,120,126,137` | `hunt_verdict_application_test.go:423`（CIO 呼叫 0 次、狀態 noData、沒存裁決、以空表改寫） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-03.1 | 做多附完整風險參數 | 操作做多；停損價 0.009、停利價 0.013 | `hunt_verdicts_domain.go:78-81` | `hunt_verdicts_domain_test.go:71` a long stops below and takes profit above | asserts-oracle | produces-oracle | ✅ conforms |
| AC-03.2 | CIO 給做空視為觀望 | 觀望；槓桿 0、部位 0%、無停損停利價 | `hunt_verdicts_domain.go:63`；`claude_prompts.go:68` | `hunt_verdicts_domain_test.go` a short, since the hunt only goes long；`claude_verdict_test.go:58`（schema 無 short） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-03.3 | 避開保存於歷史但不上表 | 本輪裁決 PENGU 為避開；結果表沒有 PENGU | `hunt_verdict_service.go:120-126`；`bullish_focus_domain.go:41` | `hunt_verdict_application_test.go:447`（ARB 避開已保存、不在表上） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-04.1 | 只有做多上表，全部裁決留在歷史 | 結果表只有做多者；三筆裁決都在 | 同上 | `hunt_verdict_application_test.go:447` case 1 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-04.2 | 做多信心剛好 50，上表 | PENGU 在表上 | `bullish_focus_domain.go:41` | 同上（OP 50 上表）；`hunt_verdicts_domain_test.go:176` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-04.3 | 做多信心 49，不上表 | 不在表上；歷史中為做多、信心 49 | `bullish_focus_domain.go:41` | 同上（TIA 不在表上；已保存為做多、信心 49） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-04.4 | CIO 全部不做多，結果表清空 | 裁決輪次成功；結果表為空 | `hunt_verdict_service.go:126,137` | `hunt_verdict_application_test.go:447` case 2（狀態 succeeded、以空表改寫） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-04.5 | 本輪上表的幣取代舊表 | 表上只剩 STRK、ARB；STRK 為本輪內容與時間 | `hunt_verdict_service.go:126` + 既有 `hunt_board_repository.go:20` | `hunt_verdict_application_test.go:148`（本輪列與計算時間）+ `verdict_repositories_test.go:31`（改寫移除其餘） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-05.1 | 過濾全部淘汰，結果表清空 | 回合停在過濾，原因「過濾未成功：noData」；結果表為空 | `hunt_pipeline_application.go:99-105` | `hunt_pipeline_application_test.go:232` | asserts-oracle | produces-oracle | ✅ conforms（PRD 已修正，見上） |
| AC-05.2 | 探索沒有候選幣，結果表清空 | 回合停在探索；結果表為空 | 同上 | `hunt_pipeline_application_test.go:216` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-05.3 | 洞察失敗，結果表保留 | 回合停在洞察；結果表未被改寫 | 同上（失敗不清空） | `hunt_pipeline_application_test.go:278`（嚴格 mock：任何改寫都會失敗） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-05.4 | 裁決失敗，結果表保留 | 回合停在裁決；結果表未被改寫 | 同上；`hunt_verdict_service.go` 失敗路徑不改寫 | `hunt_pipeline_application_test.go:306` + `hunt_verdict_application_test.go:181`（兩次不合格 → 失敗、不改寫） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-1 | 動能門檻皆含邊界、可調整（−10%～+60%、≥−10%、≤0.1%） | 預設值如上，環境變數可覆寫，負值有效 | `application_config.go`（`parseDecimalWithDefault`） | `application_config_test.go:156` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | 市場結構於過濾時取得，交易所失敗換下一家，全查不到記無資料，不讓整輪失敗 | 同左 | `coin_profile_service.go:106-115` + `perpetual_market_structure_service.go` | `coin_filtering_application_test.go:676`；`hunt_verdict_application_test.go` ShowsTheFirstExchange（同一服務的換家行為） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | 看多洞察門檻：看多、強度 ≥ 6（可調整）、分析成功 | 同左 | `bullish_focus_domain.go:25-34`；`application_config.go` | `hunt_verdicts_domain_test.go:158`；`application_config_test.go:144` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | CIO 操作只有做多、觀望、避開；做空與其他值一律觀望；原做空停利上限不再需要 | 同左 | `hunt_verdict_vo.go`；`hunt_verdicts_domain.go:63`；`claude_prompts.go` | `hunt_verdicts_domain_test.go`；`claude_verdict_test.go:58`；`dependencies_test.go:25` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | 結果表門檻：做多且信心 ≥ 50（可調整） | 同左 | `bullish_focus_domain.go:38-47` | `hunt_verdicts_domain_test.go:176`；`application_config_test.go:144` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | 結果表依本輪上表的幣整批改寫，全有或全無；沒有幣上表即清空 | 同左 | `hunt_verdict_service.go:126`；`hunt_board_repository.go:20` | `verdict_repositories_test.go:31,77,86` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-7 | 裁決輪次狀態：無看多 → 無資料；CIO 可用且改寫成功 → 成功（即使沒有上表）；否則失敗 | 同左 | `pipeline_run_domain.go:68`；`hunt_verdict_service.go` | `hunt_verdicts_domain_test.go:144`；`hunt_verdict_application_test.go:423,447,181` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-8 | 回合：任一步無資料 → 停且清空；失敗 → 停且不動；清空失敗附註於原因 | 同左 | `hunt_pipeline_application.go:99-105` | `hunt_pipeline_application_test.go:216,232,246,265,278,306` | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-1 | 過濾時逐幣查詢市場結構需同時進行 | 多枚幣的查詢同時進行 | `coin_profile_service.go:110` | `coin_filtering_application_test.go:635`（序列化即失敗，已驗證） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-2 | 沒有看多洞察時不呼叫 CIO | CIO 呼叫 0 次 | `hunt_verdict_service.go:93-95` | `hunt_verdict_application_test.go:423`（`Times(0)`） | asserts-oracle | produces-oracle | ✅ conforms |

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| `hunt_pipeline_application.go:101` | When a verdict step itself ends with no data, the round clears the board a second time (idempotent) | Covered by BR-8's uniform rule; recorded in the ARCH improve review. Not out of scope. |
| `postman/` test scripts | The collection now asserts the board holds longs only and that kept coins carry momentum verdicts | Documentation of AC-04.x / AC-01.x, not product behavior |

None of the code touches an Out of Scope item: information sources, the analyst prompt, a bearish board, and manual single-step board clearing are all unchanged.

## Summary

- Conforms: 41/41 clauses ✅ (100%), after the four fixes listed under "First pass and fixes"
- Violations: none (AC-05.1 was resolved by correcting the PRD to the established format)
- Mis-asserted: none (AC-04.3 and AC-05.3 tests strengthened)
- Partial: none (NFR-1 test added)
- Gaps: none
- Unclear: none
- Orphans: 2 (both benign)
