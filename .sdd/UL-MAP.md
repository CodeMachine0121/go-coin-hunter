# 📔 Ubiquitous Language Map

**Project:** go-coin-hunter
**Bounded Context:** 新幣獵捕（Coin Hunting）
**Maintainer:** James Hsueh
**Last Updated:** 2026-10-05

> 本文件只記錄**現在有效的詞彙**，不記錄變更歷史。詞彙不再使用就直接刪除該列。

---

## 1. Nouns & Concepts

| Domain Term | Technical Name | User-Facing Label | Definition & Business Rules | Status |
| :--- | :--- | :--- | :--- | :--- |
| 獵捕管線 | *(尚未實作)* | 獵捕管線 | 探索 → 過濾 → 洞察 → 裁決 的完整一輪 | Confirmed |
| 管線輪次 | *(尚未實作)* | 輪次 | 管線**某一個步驟**的一次執行紀錄。狀態：執行中 → 成功 / 失敗 / 無資料；記錄觸發來源與失敗原因 | Confirmed |
| 管線步驟 | *(尚未實作)* | 步驟 | 管線輪次屬於哪一步。目前只有**探索**；後續切片補上過濾、洞察、裁決 | Confirmed |
| 觸發來源 | *(尚未實作)* | 觸發 | 輪次由**排程**還是**手動**發動 | Confirmed |
| 輪次狀態 | *(尚未實作)* | 狀態 | **執行中**、**成功**、**失敗**、**無資料**。「無資料」指乾淨跑完但沒有產出任何候選幣 | Confirmed |
| 資訊來源 | *(尚未實作)* | 來源 | 一個免費的外部情報出處（幣安公告、幣安永續合約、Bybit 公告、OKX 公告、CoinGecko 熱門、DEX Screener 新幣看板）。可增減；彼此獨立，一個失敗不影響其他 | Confirmed |
| 來源結果 | *(尚未實作)* | 來源狀態 | 某一輪中某一個資訊來源的成敗、取回幾則情報、失敗原因 | Confirmed |
| 情報 | *(尚未實作)* | 情報 | 某個資訊來源發布的一則訊息：來源、提到的幣種代號（可能沒有）、標題、原文連結、發布時間。同來源同一則只保存一份 | Confirmed |
| 幣種代號 | *(尚未實作)* | 代號 | 一枚幣的交易代號（如 CT、PUMP），一律大寫。不同鏈上同代號視為同一枚幣 | Confirmed |
| 傳統金融商品 | *(尚未實作)* | — | 交易所上架的代幣化股票、股票永續合約等非加密原生資產。**不是新幣**，不成為候選幣 | Confirmed |
| 排除幣種 | *(尚未實作)* | 排除名單 | 主流幣與穩定幣（預設 BTC、ETH、BNB、SOL、XRP、USDT、USDC、FDUSD、DAI、TUSD、USDE）。**不是新幣**，不成為候選幣；名單可調整 | Confirmed |
| 探索時間窗 | *(尚未實作)* | 時間窗 | 只有發布時間落在「本輪開始往前 N 小時」內（含邊界）的情報才進入本輪候選幣。預設 72 小時 | Confirmed |
| 候選幣 | *(尚未實作)* | 候選幣 | 一輪探索中，時間窗內被提到、且不屬排除幣種或傳統金融商品的一枚幣。每輪每個代號只有一個，記下提到它的來源數、情報數、最早被提到的時間 | Confirmed |

## 2. Actions & Processes

| Business Action | Technical Name | Definition | Status |
| :--- | :--- | :--- | :--- |
| 探索 | *(尚未實作)* | 向全部資訊來源收集情報、保存、彙出本輪候選幣 | Confirmed |
| 中斷收尾 | *(尚未實作)* | 服務啟動時，把殘留「執行中」的輪次改為「失敗（被重啟中斷）」 | Confirmed |
