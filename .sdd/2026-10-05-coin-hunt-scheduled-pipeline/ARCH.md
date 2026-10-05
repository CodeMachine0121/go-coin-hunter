# 獵捕管線排程 — Architecture Design

**Status:** Confirmed（使用者授權一律採 best practice）
**Tech context:** Go · `time.Ticker` + goroutine · Clean / Onion

## 1. Design Goal & Guiding Principle
- 跨四個 Domain Service 的編排屬於 **Application 層**：`HuntPipelineApplication.RunHuntRound(ctx, trigger)` 依序呼叫四個 service，決定何時停。
- 排程是機制，不是規則：`HuntPipelineJob` 只負責「何時跑、不重疊、可停止」，透過 `IHuntPipelineApplication` 呼叫回合，因此 job 可用 mock 單獨測試。

## 2. Change Scope / New Components

| Name | Kind | Responsibility |
| :--- | :--- | :--- |
| `IHuntPipelineApplication`（`domain/interface/`） | 介面 | `RunHuntRound(ctx, trigger) dto.HuntRoundDto` |
| `HuntPipelineApplication`（`application/`） | Application | 依序 `DiscoverCoins` → `FilterCoinCandidates` → `AnalyzeCoinCandidates` → `SynthesizeHuntVerdicts`；任一錯誤或非成功狀態即停；回報 `HuntRoundDto{Steps []PipelineRunDto, Completed, StoppedStep, StoppedReason}`；`ctx` 已結束時不開始下一步 |
| `HuntRoundDto` | DTO | 回合結果 |
| `HuntPipelineJob`（`job/`） | Background job | `Start`：goroutine 立即跑一輪，之後每個 interval 一輪（`time.Ticker`）；`atomic.Bool` 防重疊（上一輪未完即跳過）；`Stop`：停止 ticker、不再開新回合並等進行中的回合在步驟邊界結束；`Finished`：goroutine 結束時關閉；每輪記日誌 |
| `HuntPipelineController` | Controller | `POST /hunt-rounds`：手動一鍵（觸發來源 `manual`），回 `HuntRoundDto` |
| `config.PipelineConfig` | 設定 | `HUNT_PIPELINE_INTERVAL_HOURS`（預設 4，`≤0` 停用） |
| `dependencies.backgroundJobsFor` | 組裝 | 總開關開且間隔 > 0 → `[HuntPipelineJob]` |

## 3. Extensibility
- 新增第五步 = `HuntPipelineApplication` 多一段呼叫；推播可在回合完成後加入。
- 多副本時需改為租約式領導權（go-trading 的 job leadership），目前單機不需要。

## 4. Traceability

| Scenario | Fulfilled by |
| :--- | :--- |
| 四步成功 / 各步未成功即停 / 出錯即停 / 裁決失敗 | `HuntPipelineApplication.RunHuntRound` |
| 啟動即跑並重複 / 間隔 0 / 總開關 / 跳過重疊 / 關閉時停 | `HuntPipelineJob` + `backgroundJobsFor` |
| 手動一鍵 | `HuntPipelineController` |
