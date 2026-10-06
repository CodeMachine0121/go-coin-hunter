# Roadmap — 功能切片規劃

整條獵捕管線拆成 5 個切片，之後再以切片 6 收斂為「只找做多」，依序實作；每個切片各走完 `clarify → prd → architecture → implement → improve-codebase → contract`（見 `.claude/rules/workflow.md`）。

| # | 切片 | 範圍 | 狀態 |
| :-- | :--- | :--- | :--- |
| 1 | `coin-intelligence-discovery` | `IInformationSourceProxy` 抽象 + 六個免費資訊來源實作（DI 注入介面 list）→ 情報落地、彙出候選幣；管線輪次記錄基礎；手動觸發 API | 完成 |
| 2 | `coin-candidate-filtering` | `ICoinCandidateFilterHandler` 策略模式：安全檢查、流動性門檻、FDV、流通比、解鎖時程、是否已上永續合約，各一個 Handler；過濾結果與淘汰理由落地 | 完成 |
| 3 | `coin-insight-analysis` | Claude 依情報 + 市場結構（OI / 資金費率 / 成交量）萃取每檔候選幣訊號，AI 互動 1–2 輪 | 完成 |
| 4 | `coin-hunt-verdict` | CIO 綜合洞察 → 最終裁決；**獵捕結果表**每輪覆寫（本輪有的覆蓋、本輪沒有的移除） | 完成 |
| 5 | `coin-hunt-scheduled-pipeline` | 背景 job 依序串起 ①→④，上游失敗即中止本輪；查詢 API 與 Postman 集合 | 完成 |
| 6 | `bullish-coin-hunt` | 只獵看好的幣：過濾加三條動能規則（漲跌幅、持倉量變化、資金費率過熱）；只把看多且夠強的洞察交給 CIO、CIO 不做空；獵捕結果表只放做多且信心達標的幣，回合無資料時清空 | 完成 |

## 之後可以做的事

- 設定 `ANTHROPIC_API_KEY` 後實際跑一輪，檢視洞察與 CIO 裁決的品質，再調整系統提示與思考深度。
- 獵捕結果表變化時推播（go-trading 已有 Telegram 推送能力）。
- 回測：以 `CoinVerdict` 歷史比對裁決後的實際價格表現，校準信心。
- 更多免費資訊來源（Upbit 公告、GeckoTerminal 新池）：實作 `IInformationSourceProxy` 並在組裝根多一行。
