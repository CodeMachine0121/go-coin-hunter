# Ubiquitous Language Map — go-coin-hunter

所有實體 / 動作 / 識別字命名以此為準，不得自創同義詞。各切片的 `/clarify`、`/prd` 會持續補充本表。

## Bounded Context: Coin Hunting（新幣獵捕）

| 業務詞彙 | 程式識別字 | 說明 |
| :--- | :--- | :--- |
| 獵捕管線 | `HuntPipeline` | 探索 → 過濾 → 洞察 → 裁決 的完整一輪 |
| 管線輪次 | `PipelineRun` | 管線某一步驟的一次執行記錄 |
