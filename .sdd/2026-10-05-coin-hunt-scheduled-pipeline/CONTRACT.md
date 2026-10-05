# 契約追溯矩陣 — 獵捕管線排程（coin-hunt-scheduled-pipeline）

Contract: PRD.md（v1.0, Finalized；含 `b4475c6` 更新的 Section 4）
Design map: ARCH.md
Implementation: `internal/application/hunt_pipeline_application.go`、`internal/job/hunt_pipeline_job.go`、`internal/controller/hunt_pipeline_controller.go`、`internal/domain/models/domains/pipeline_run_step_domain.go`、`internal/domain/models/domains/hunt_round_errors.go`、`cmd/server/dependencies.go`、`cmd/server/main.go`、`internal/config/application_config.go`
Oracle: 驗收條件（23 條：AC 11、BR 11、NFR 1）
Audited at: HEAD `79cee58`（含 `78a78a4` 修正、`b4475c6` PRD 更新）。本次從頭重跑，取代前一版矩陣。

> 上限聲明：本文件為**靜態**契約符合度稽核。以 PRD 推導的 oracle 分別比對測試斷言與正式程式碼路徑；不撰寫新探針、不執行自創情境。僅執行了各條款既有對應測試作為佐證（全部通過，`-count=1 -timeout 120s`，未呼叫真實 Anthropic API），判定不以通過與否為準。

## Out of Scope（反向檢核清單）

- 推播
- 多副本協調
- 依時段調整間隔

檢核結果：程式碼中未發現上述任何一項的實作（重疊防護為單一行程內的 `atomic.Bool`，非多副本協調）。

## Clauses

`Spec-expected` 欄為 Phase 2 僅依 PRD 文字推導的業務可觀察 oracle；稽核欄位檢查的是經 UL-MAP/ARCH 橋接後的具體產物（`HuntRoundDto{Steps, Completed, StoppedStep, StoppedReason}`、`PipelineRun.Status` = `succeeded`/`failed`/`noData`、`TriggerSource` = `job`/`manual`、`ErrHuntRoundAlreadyRunning`、HTTP 200/409）。

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | Scenario: 四步都成功 — Given 探索、過濾、洞察、裁決都會成功 / When 執行一輪獵捕回合 / Then 四步依序各執行一次 / And 回合為「完成」 | 探索→過濾→洞察→裁決各恰好執行一次且順序正確；回合標記為完成 | `hunt_pipeline_application.go:67-75,78-102` | `hunt_pipeline_application_test.go:174` TestRunHuntRoundRunsEveryStepInOrder | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | Scenario: 探索無資料即停止 — Given 探索的輪次為「無資料」/ When 執行一輪獵捕回合 / Then 只執行了探索 / And 回合停在探索，原因為「探索未成功：noData」 | 僅探索被執行；回合未完成、停在探索、原因恰為「探索未成功：noData」 | `hunt_pipeline_application.go:97-99`；`pipeline_run_step_domain.go:36-37` | `hunt_pipeline_application_test.go:203` discovery with no data | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | Scenario: 過濾無資料即停止 — Given 探索成功、過濾的輪次為「無資料」/ When 執行一輪獵捕回合 / Then 洞察與裁決都沒有執行 / And 回合停在過濾，原因為「過濾未成功：noData」 | 洞察、裁決皆未執行；回合停在過濾，原因恰為「過濾未成功：noData」 | `hunt_pipeline_application.go:97-99` | `hunt_pipeline_application_test.go:217` filtering keeping nothing | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | Scenario: 某一步出錯即停止 — Given 探索與過濾成功、洞察執行時出錯「disk full」/ When 執行一輪獵捕回合 / Then 裁決沒有執行 / And 回合停在洞察，原因包含「disk full」 | 裁決未執行；回合停在洞察，原因文字含「disk full」 | `hunt_pipeline_application.go:91-95` | `hunt_pipeline_application_test.go:230` insight failing as a step error | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | Scenario: 裁決失敗也記為未完成 — Given 前三步成功、裁決的輪次為「失敗」/ When 執行一輪獵捕回合 / Then 回合停在裁決，原因為「裁決未成功：failed」 | 回合未完成、停在裁決，原因恰為「裁決未成功：failed」 | `hunt_pipeline_application.go:97-99` | `hunt_pipeline_application_test.go:245` verdict failing | asserts-oracle | produces-oracle | ✅ conforms |
| AC-6 | Scenario: 啟動後立即執行並依間隔重複 — Given 排程間隔為 4 小時 / When 服務啟動 / Then 立刻執行一輪回合，觸發來源為「排程」/ And 每隔 4 小時再執行一輪 | 啟動當下（不等待間隔）即跑一輪、觸發來源為排程；其後每輪相隔一個間隔再跑 | `hunt_pipeline_job.go:38-44,61-62`；`dependencies.go:265`；`application_config.go:162` | `hunt_pipeline_job_test.go:28` TestHuntPipelineJobRunsAtStartThenEveryInterval；`application_config_test.go:62` | asserts-oracle（首輪 `< interval/2`、觸發來源 `job`、相鄰兩輪間距 `≥ 3/4 interval`；預設 4 小時由 config 測試釘住） | produces-oracle | ✅ conforms |
| AC-7 | Scenario: 間隔為 0 即停用 — Given 排程間隔為 0 / When 服務啟動 / Then 不排程任何回合 | 不會有任何排程回合 | `dependencies.go:260-262`；`main.go:54-55` | `dependencies_test.go:63` a zero interval | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | Scenario: 背景作業總開關關閉即停用 — Given 背景作業總開關為關閉 / When 服務啟動 / Then 不排程任何回合 | 不會有任何排程回合 | `dependencies.go:261`；`application_config.go:135` | `dependencies_test.go:63` background jobs switched off；`application_config_test.go:30` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | Scenario: 上一輪未結束時跳過 — Given 上一輪回合仍在執行 / When 下一個排程時間點到了 / Then 這一次不開始新的回合 | 上一輪未結束期間，時間點到也不會開始第二輪 | `hunt_pipeline_application.go:44-47`；`hunt_pipeline_job.go:61-64` | `hunt_pipeline_application_test.go:306` TestRunHuntRoundRefusesASecondRoundWhileOneIsRunning；`hunt_pipeline_job_test.go:62` | asserts-oracle（共用防護：第二輪被拒、未多跑任何步驟；job 被拒後仍維持排程） | produces-oracle（job 與手動路由共用同一個 `builtApplications.huntPipeline` 實例，`dependencies.go:252,265`） | ✅ conforms |
| AC-10 | Scenario: 關閉時做完目前這一步就停 — Given 回合正在執行洞察 / When 服務收到關閉指令 / Then 洞察完成後回合停止 / And 裁決沒有執行 | 進行中的步驟不被中斷而完成；其後回合停止，下一步（裁決）不執行 | `hunt_pipeline_application.go:80-85`；`hunt_pipeline_job.go:61-62,68-70`；`main.go:68` | `hunt_pipeline_application_test.go:284`；`hunt_pipeline_job_test.go:80` | asserts-oracle（應用層測試以「探索進行中」代替「洞察進行中」；迴圈對各步一致，屬等價） | produces-oracle | ✅ conforms |
| AC-11 | Scenario: 手動執行回報每一步 — Given 四步都會成功 / When 交易者手動執行一輪獵捕回合 / Then 每一步的觸發來源為「手動」/ And 回報依序包含四步的輪次與回合「完成」 | 四步的輪次觸發來源皆為手動；回報中依序列出四步輪次，並標示回合完成 | `hunt_pipeline_controller.go:25-33`；`dependencies.go:252-253` | `hunt_pipeline_controller_test.go:22`；`hunt_pipeline_application_test.go:245`、`:174` | asserts-oracle（組合式：controller 以 `manual` 呼叫並原樣回傳 DTO；`:245` 釘住 `manual` 傳到每一步；`:174` 釘住四步依序與完成。無單一測試以手動觸發斷言完整組合） | produces-oracle | ✅ conforms |
| BR-1 | 回合：探索 → 過濾 → 洞察 → 裁決，同一個執行緒內依序；每步用該步自己的規則與上游的「最新成功輪次」，因依序執行，上游即本回合剛產出者。 | 四步在同一執行緒依序執行；每步取用的上游即本回合前一步剛產出的成功輪次 | `hunt_pipeline_application.go:78-101`；`:44-47`（防重疊確保無他輪交錯） | `hunt_pipeline_application_test.go:174`（`TriggeredByPipelineRunID` 串接）、`:306` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | 停止條件：某步回傳錯誤 → 停在該步，原因為錯誤訊息 | 出錯的那一步即停止點，原因為該錯誤訊息 | `hunt_pipeline_application.go:92-94` | `hunt_pipeline_application_test.go:230` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | 某步輪次狀態不是「成功」→ 停在該步，原因「{步驟}未成功：{狀態}」 | 非成功狀態的那一步即停止點，原因為「{步驟中文名}未成功：{狀態}」 | `hunt_pipeline_application.go:97-99`；`pipeline_run_step_domain.go:36-37` | `hunt_pipeline_application_test.go:203,217,245`；`pipeline_run_step_domain_test.go:11` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | 服務關閉中或已被放棄而不開始某步 → 停在該步，原因「{步驟}未開始：{原因}」。 | 因關閉或被放棄而未開始的那一步即停止點，原因為「{步驟中文名}未開始：{原因}」 | `hunt_pipeline_application.go:80-89`；`pipeline_run_step_domain.go:31-32` | `hunt_pipeline_application_test.go:261`（探索未開始：服務關閉中）、`:274`（探索未開始：context canceled）、`:284`（過濾未開始：服務關閉中）；`pipeline_run_step_domain_test.go:20` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | 同一時間至多一輪（排程與手動共用）：已有回合進行中時，排程的這一次跳過（日誌記「hunt round skipped」）、手動被告知「已有獵捕回合進行中」。 | 任一時刻至多一輪；排程撞上時本次不跑且留下「hunt round skipped」日誌；手動撞上時被告知「已有獵捕回合進行中」 | `hunt_pipeline_application.go:44-47`；`hunt_round_errors.go:6`；`hunt_pipeline_job.go:61-64`；`hunt_pipeline_controller.go:28-31` | `hunt_pipeline_application_test.go:306`；`hunt_pipeline_job_test.go:62`；`hunt_pipeline_controller_test.go:42` | **shallow**（僅日誌部分）：共用防護、拒絕訊息「已有獵捕回合進行中」、手動 409 皆有斷言；但**沒有任何測試斷言排程被拒時會記「hunt round skipped」**——`hunt_pipeline_job_test.go:62` 只斷言排程持續，移除 `hunt_pipeline_job.go:63` 的日誌測試仍通過 | produces-oracle（`log.Printf("hunt round skipped: %v", ...)`） | 🟠 mis-asserted |
| BR-6 | 排程：間隔預設 4 小時（`0` 或負值停用） | 未設定時間隔為 4 小時；設 0 或負值則不排程 | `application_config.go:162`；`dependencies.go:261` | `application_config_test.go:62`；`dependencies_test.go:63` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-7 | 排程：啟動即跑第一輪。 | 服務啟動時不等待間隔即跑第一輪 | `hunt_pipeline_job.go:40` | `hunt_pipeline_job_test.go:28`（`:56`） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-8 | 關閉：排程與手動回合都不再開始下一步 | 關閉開始後，不論排程或手動回合，都不再開始任何新步驟 | `hunt_pipeline_application.go:80-85`；`hunt_pipeline_job.go:61-62,68-70`；`hunt_pipeline_controller.go:26-27`；`main.go:48,68` | `hunt_pipeline_application_test.go:284`；`hunt_pipeline_job_test.go:80`；`hunt_pipeline_controller_test.go:22`（斷言手動回合拿到關閉通道） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-9 | 進行中的一步最多等**關閉寬限期**（預設 15 分鐘，可調整），逾時即放棄，該步的輪次於下次啟動時標為「失敗（被重啟中斷）」。 | 關閉時最多等寬限期（預設 15 分鐘、可設定）讓進行中的一步結束；逾時即放棄；被放棄那一步的輪次在下次啟動時成為「失敗（被重啟中斷）」 | `application_config.go:166`；`main.go:65-75`；`pipeline_run_service.go:39-47`；`main.go:36` | `application_config_test.go:72`（預設 15、可覆寫）；`pipeline_run_application_test.go:31`（掃描為 failed／被重啟中斷） | no-test（`main.go:65-75` 的「等到寬限期、逾時放棄」順序無測試；只有設定值與重啟掃描各自有測試） | produces-oracle（`server.Shutdown` 與 `WaitAll` 共用同一個寬限期 deadline，因重疊防護同時至多一輪，總等待 ≤ 寬限期；逾時 `abandonJobs()` 後 `main` 返回，輪次停在 running，下次啟動 `main.go:36` 掃為「被重啟中斷」） | 🟡 partial |
| BR-10 | 紀錄：每輪回合（排程與手動）結束記一筆日誌：觸發來源、完成與否、停在哪一步與原因、各步已產出的輪次編號；出錯的步驟以步驟名稱與原因記錄，其失敗輪次可在輪次歷史查到。 | 每輪（排程或手動）結束各留一筆日誌，含觸發來源、是否完成、停止步驟與原因、各步已產出輪次編號；出錯步驟以名稱＋原因出現 | `hunt_pipeline_application.go:49-58`（所有觸發來源共用） | — | no-test（沒有任何測試擷取或斷言日誌內容） | produces-oracle（`hunt round (%s) completed: pipeline runs %v` / `hunt round (%s) stopped at %s (%s): pipeline runs %v`；出錯步驟以 `StoppedStep`＋錯誤訊息出現；其失敗輪次由各步 service 落地，如 `coin_insight_service.go:136-142`） | 🟡 partial |
| BR-11 | 手動回應：回合跑完或中途停止都以正常結果回應（附每一步與停止原因）；只有已有回合進行中時回應衝突。 | 手動觸發：完成或停止 → 正常結果，含各步輪次與停止原因；只有撞上進行中回合時回衝突 | `hunt_pipeline_controller.go:25-33` | `hunt_pipeline_controller_test.go:22`（停止回合 → 200＋完整內容）、`:42`（→ 409「已有獵捕回合進行中」） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-1 | 回合本身不額外呼叫付費服務（費用即各步自身的 AI 呼叫）。 | 跑一輪的付費 AI 呼叫數＝四步自身呼叫數之和，回合編排不另外呼叫 | `hunt_pipeline_application.go:67-101`（只呼叫四個 service） | `hunt_pipeline_application_test.go:174`（gomock 預設 `Times(1)`：analyst、strategist 各恰一次） | asserts-oracle | produces-oracle | ✅ conforms |

## Orphans（沒有任何條款解釋的行為）

| Code | Description | Verdict |
|------|-------------|---------|
| `hunt_pipeline_controller.go:26-27` | 手動回合使用 HTTP 請求的 context：交易者的用戶端斷線（如 curl 逾時）時，進行中的那一步被取消，後續步驟以「{步驟}未開始：context canceled」停止。PRD 的「未開始」只涵蓋「服務關閉中或已被放棄」，未提用戶端斷線 | undocumented |
| `application_config.go:166` | `SHUTDOWN_GRACE_MINUTES` 設 0、負值或打錯時一律退回 15 分鐘（`parsePositiveIntWithDefault`）。PRD 只說「可調整」，未定義非正值的意義（與間隔設 0 即停用的慣例不同） | undocumented |

## Summary

- Conforms: 20/23 clauses ✅（87%）
- Violations: —
- Mis-asserted: BR-5（排程跳過的「hunt round skipped」日誌未被斷言）
- Partial: BR-9（`main.go` 寬限期等待／放棄順序無測試）、BR-10（每輪日誌內容無測試）
- Gaps: —
- Unclear: —
- Orphans: 2（皆非 Out of Scope）

前一版的三項違規（重疊防護只在 job、關閉固定 4 分鐘截斷、手動回合無日誌／漏記出錯步驟）在 `78a78a4` 與更新後的 PRD 下已不再成立。

佐證執行（`-count=1 -timeout 120s`，皆通過）：`TestRunHuntRound*`（含 `TestRunHuntRoundRefusesASecondRoundWhileOneIsRunning`）、`TestHuntPipelineJob*`、`TestHuntRoundRoute`、`TestPipelineRunStepLabels`、`TestTheHuntRoundIsScheduledOnlyWhenSwitchedOn`、`TestLoadReadsTheHuntPipelineInterval`、`TestLoadReadsTheShutdownGracePeriod`、`TestLoadReadsOverrides`、`TestFailInterruptedPipelineRuns*`。

---

## Resolution（第二輪稽核後處理，2026-10-05）

| Clause / 項目 | 原狀態 | 處理 | 驗證 |
| :--- | :--- | :--- | :--- |
| BR-5 | 🟠 mis-asserted | job 測試擷取日誌，斷言「hunt round skipped: 已有獵捕回合進行中」（commit `7fe40f8`） | 拿掉跳過日誌的 mutation 被抓到 |
| BR-9 | 🟡 partial | 關閉流程自 `main` 抽出為 `shutDown`，測試「寬限內等到 job 結束不放棄」與「寬限用盡即放棄」 | 拿掉放棄、拿掉要求停止的 mutation 均被抓到 |
| BR-10 | 🟡 partial | application 測試擷取日誌，斷言完成與停止兩種日誌的觸發來源、停止步驟與原因、輪次編號 | 改寫完成日誌內容的 mutation 被抓到 |
| Orphan 1（手動回合跟著請求） | ⚠️ | PRD 已註明：呼叫端斷線即放棄進行中步驟，原因「{步驟}未開始：context canceled」 | — |
| Orphan 2（寬限期非正值） | ⚠️ | PRD 已註明：0、負值或寫錯一律採預設 15 分鐘 | — |

處理後：**23 conforms · 0 violation · 0 mis-asserted · 0 partial · 0 orphan**。全套測試（含 PostgreSQL 儲存層、-race）通過，`internal/` 非 mock 程式碼覆蓋率 100%。
