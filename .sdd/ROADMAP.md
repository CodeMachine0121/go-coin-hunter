# Roadmap — 功能切片規劃

整條獵捕管線拆成 5 個切片，依序實作；每個切片各走完 `clarify → prd → architecture → implement → improve-codebase → contract`（見 `.claude/rules/workflow.md`）。

| # | 切片 | 範圍 | 狀態 |
| :-- | :--- | :--- | :--- |
| 1 | `coin-intelligence-discovery` | `IInformationProvider` 抽象 + 免費資訊來源 Provider 清單（DI 注入介面 list）→ 情報落地、彙出候選幣；管線輪次記錄基礎；手動觸發 API | 完成 |
| 2 | `coin-candidate-filtering` | `ICoinCandidateFilter` 策略模式：安全檢查、流動性門檻、FDV、流通比、解鎖時程、是否已上永續合約，各一個 Handler；過濾結果與淘汰理由落地 | 規劃中 |
| 3 | `coin-insight-analysis` | Claude 依情報 + 市場結構（OI / 資金費率 / 成交量）萃取每檔候選幣訊號，AI 互動 1–2 輪 | 規劃中 |
| 4 | `coin-hunt-verdict` | CIO 綜合洞察 → 最終裁決；**獵捕結果表**每輪覆寫（本輪有的覆蓋、本輪沒有的移除） | 規劃中 |
| 5 | `coin-hunt-scheduled-pipeline` | 背景 job 依序串起 ①→④，上游失敗即中止本輪；查詢 API 與 Postman 集合 | 規劃中 |
