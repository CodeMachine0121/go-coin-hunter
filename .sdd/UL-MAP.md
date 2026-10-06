# 📔 Ubiquitous Language Map

**Project:** go-coin-hunter
**Bounded Context:** 新幣獵捕（Coin Hunting）
**Maintainer:** James Hsueh
**Last Updated:** 2026-10-06

> 本文件只記錄**現在有效的詞彙**，不記錄變更歷史。詞彙不再使用就直接刪除該列。

---

## 1. Nouns & Concepts

| Domain Term | Technical Name | User-Facing Label | Definition & Business Rules | Status |
| :--- | :--- | :--- | :--- | :--- |
| 獵捕管線 | `HuntPipelineApplication` | 獵捕管線 | 探索 → 過濾 → 洞察 → 裁決 四個步驟 | Confirmed |
| 獵捕回合 | `HuntRoundDto` | 回合 | 依序執行四個步驟的一輪；任一步未成功即停在該步；由排程（每 4 小時）或手動一鍵觸發 | Confirmed |
| 管線輪次 | `PipelineRun` | 輪次 | 管線**某一個步驟**的一次執行紀錄。狀態：執行中 → 成功 / 失敗 / 無資料；記錄觸發來源與失敗原因 | Confirmed |
| 管線步驟 | `PipelineRunStepVo` | 步驟 | 管線輪次屬於哪一步：**探索**、**過濾**、**洞察**、**裁決** | Confirmed |
| 觸發來源 | `PipelineRunTriggerSourceVo` | 觸發 | 輪次由**排程**還是**手動**發動 | Confirmed |
| 輪次狀態 | `PipelineRunStatusVo` | 狀態 | **執行中**、**成功**、**失敗**、**無資料**。「無資料」指乾淨跑完但沒有東西可交給下一步：探索沒有候選幣、過濾沒有保留的幣、裁決沒有看多洞察可交給 CIO | Confirmed |
| 資訊來源 | `IInformationSourceProxy` | 來源 | 一個免費的外部情報出處（幣安公告、幣安永續合約、Bybit 公告、OKX 公告、CoinGecko 熱門、DEX Screener 新幣看板）。可增減；彼此獨立，一個失敗不影響其他 | Confirmed |
| 來源結果 | `InformationSourceOutcome` / `InformationSourceResultsDomain` | 來源狀態 | 某一輪中某一個資訊來源的成敗、取回幾則情報、失敗原因 | Confirmed |
| 情報 | `CoinIntelligence` | 情報 | 某個資訊來源發布的一則訊息：來源、提到的幣種代號（可能沒有）、標題、原文連結、發布時間。同來源同一則只保存一份 | Confirmed |
| 幣種代號 | `CoinSymbol` | 代號 | 一枚幣的交易代號（如 CT、PUMP），一律大寫。不同鏈上同代號視為同一枚幣 | Confirmed |
| 傳統金融商品 | `IsTraditionalAsset` | — | 交易所上架的代幣化股票、股票永續合約等非加密原生資產。**不是新幣**，不成為候選幣 | Confirmed |
| 排除幣種 | `DiscoveryPolicyVo.ExcludedCoinSymbols` | 排除名單 | 主流幣與穩定幣（預設 BTC、ETH、BNB、SOL、XRP、USDT、USDC、FDUSD、DAI、TUSD、USDE）。**不是新幣**，不成為候選幣；名單可調整 | Confirmed |
| 過濾規則 | `ICoinCandidateFilterHandler` | 規則 | 判斷一枚候選幣能否留下的一條獨立規則：安全檢查、流動性門檻、完全稀釋估值、流通比、解鎖時程、是否已上永續合約，以及三條動能規則（24 小時漲跌幅、持倉量變化、資金費率過熱）。可增減，彼此獨立 | Confirmed |
| 規則結果 | `FilterVerdictVo` | 結果 | 一條過濾規則對一枚候選幣的判斷：**通過**、**淘汰**、**無資料**；淘汰與無資料必附理由 | Confirmed |
| 過濾結果 | `CoinFilterResult` | 過濾結果 | 一輪過濾中一枚候選幣的全部規則結果與是否保留。任一規則淘汰即淘汰 | Confirmed |
| 保留的候選幣 | `CoinFilterResult.IsKept` | 保留 | 沒有任何過濾規則淘汰的候選幣，交給後續 AI 洞察 | Confirmed |
| 完全稀釋估值 | `CoinMarketDataVo.FullyDilutedValuationUsd` | FDV | 以總量計算的市值（美元）。預設須介於 1,000 萬～10 億美元（含兩端） | Confirmed |
| 流通比 | `CirculatingRatioFilterHandler` | 流通比 | 流通量 ÷ 最大供給量（無最大供給量改用總供給量）。預設至少 20% | Confirmed |
| 24 小時成交額 | `CoinMarketDataVo.DailyVolumeUsd` | 成交額 | 最近 24 小時成交金額（美元），流動性門檻依據。預設至少 100 萬美元 | Confirmed |
| 解鎖時程 | `TokenUnlockEventVo` | 解鎖 | 預定釋出代幣的時間與數量。預設 14 天內累計解鎖達流通量 5% 即淘汰 | Confirmed |
| 永續合約上架 | `IPerpetualContractListingProxy` | 永續合約 | 幣安、Bybit、OKX 任一家有以 USDT 計價、可交易的永續合約 | Confirmed |
| 幣種檔案 | `CoinProfileVo` | — | 過濾一枚候選幣所需的資料彙整：市值資料、合約位址、安全資料、永續合約上架、解鎖時程、此刻的永續合約市場結構 | Confirmed |
| 動能規則 | `PriceChangeFilterHandler` / `OpenInterestChangeFilterHandler` / `FundingRateOverheatFilterHandler` | 動能 | 依此刻永續合約市場結構，在 AI 之前淘汰不利做多的候選幣的三條過濾規則；查不到市場結構或該數字時記「無資料」、不淘汰 | Confirmed |
| 24 小時漲跌幅規則 | `PriceChangeFilterHandler` | 漲跌幅 | 24 小時漲跌幅須介於下限與上限之間（含兩端），預設 −10% ～ +60%：跌太多是下跌中，漲太多是追高 | Confirmed |
| 持倉量變化規則 | `OpenInterestChangeFilterHandler` | 持倉變化 | 持倉量 24 小時變化須至少達下限（含），預設 −10%：減少更多代表資金撤離 | Confirmed |
| 資金費率過熱規則 | `FundingRateOverheatFilterHandler` | 資金費率 | 資金費率先依結算週期折合成每 8 小時，須不高於上限（含），預設 0.1%：更高代表多方過度擁擠；負費率不淘汰 | Confirmed |
| 資金費率結算週期 | `PerpetualMarketStructureVo.FundingIntervalHours` | 結算週期 | 永續合約每幾小時結算一次資金費率（8、4 或 1）；同樣的每期費率，週期越短代表越擁擠。查不到時視為 8 小時 | Confirmed |
| 洞察 | `CoinInsight` | 洞察 | AI 對一枚保留候選幣的結構化判斷：訊號方向、訊號強度、催化劑、主要風險、關鍵證據、資料缺口；或「分析失敗」及原因 | Confirmed |
| 訊號方向 | `CoinInsightDirectionVo` | 方向 | **看多**、**看空**、**中性**；AI 回覆其他值一律中性 | Confirmed |
| 訊號強度 | `CoinInsight.Strength` | 強度 | 1–10 的整數；超出夾回邊界 | Confirmed |
| 永續合約市場結構 | `PerpetualMarketStructureVo` | 市場結構 | 最新價格、24 小時漲跌幅、24 小時成交額、資金費率、持倉量與其 24 小時變化；依幣安 → Bybit → OKX 取第一家有合約的 | Confirmed |
| 資料缺口 | `CoinInsight.DataGaps` | 缺口 | 分析一枚幣時缺少的素材（查不到新聞、查不到市場結構，或 AI 自行回報的缺口）；不擋分析 | Confirmed |
| 洞察素材 | `CoinInsightMaterialVo` | — | 交給 AI 的一枚幣的全部輸入：情報標題、新聞標題、市場結構、過濾結果 | Confirmed |
| 看多洞察 | `HuntVerdictPolicyVo.MinimumBullishInsightStrength` | 看多 | 分析成功、方向看多、且訊號強度達門檻（含，預設 6）的洞察；只有看多洞察會交給 CIO | Confirmed |
| 裁決 | `CoinVerdict` | 裁決 | CIO 對一枚看多洞察的幣的操作建議：操作、信心、建議槓桿、部位大小、停損與停利（距離與價格）、理由、矛盾取捨說明。每輪全部裁決保存於歷史 | Confirmed |
| 操作 | `HuntActionVo` | 操作 | **做多** long、**觀望** watch、**避開** avoid；不做空，做空與其他值一律觀望 | Confirmed |
| 信心 | `CoinVerdict.Confidence` | 信心 | 0–100；超出夾回 | Confirmed |
| 建議槓桿 | `CoinVerdict.Leverage` | 槓桿 | 1–5 倍（做多）；觀望 / 避開為 0 | Confirmed |
| 部位大小 | `CoinVerdict.PositionSizeRatio` | 部位 | 佔總資金 0%–10%；觀望 / 避開為 0 | Confirmed |
| 停損距離 / 停利距離 | `CoinVerdict.StopLossRatio` / `CoinVerdict.TakeProfitRatio` | 停損 / 停利 | 以最新價格百分比表示：停損 1%–50%、停利 1%–200%；系統換算成停損價（在下）/ 停利價（在上） | Confirmed |
| 參考價格 | `CoinVerdict.ReferencePrice` | 參考價 | 做多裁決換算停損停利所用的最新價格 | Confirmed |
| 矛盾取捨說明 | `CoinVerdict.ConflictResolution` | 取捨 | 洞察之間或洞察與市場結構矛盾時，CIO 採信哪一邊與理由 | Confirmed |
| 獵捕結果表 | `HuntBoardEntry` | 獵捕結果 | 只放**上表的幣**，每枚一列（編號、幣種代號、計算時間＋裁決內容）；每輪裁決成功或無資料時依本輪上表的幣改寫：本輪有的覆蓋、沒有的移除，全有或全無；獵捕回合任一步以「無資料」結束也清空；失敗不動 | Confirmed |
| 上表 | `HuntVerdictPolicyVo.MinimumHuntBoardConfidence` | 上表 | 裁決為做多且信心達門檻（含，預設 50）才放上獵捕結果表；其餘裁決只留在歷史 | Confirmed |
| 探索時間窗 | `DiscoveryPolicyVo.Window` | 時間窗 | 只有發布時間落在「本輪開始往前 N 小時」內（含邊界）的情報才進入本輪候選幣。預設 72 小時 | Confirmed |
| 候選幣 | `CoinCandidate` | 候選幣 | 一輪探索中，時間窗內被提到、且不屬排除幣種或傳統金融商品的一枚幣。每輪每個代號只有一個，記下提到它的來源數、情報數、最早被提到的時間 | Confirmed |

## 2. Actions & Processes

| Business Action | Technical Name | Definition | Status |
| :--- | :--- | :--- | :--- |
| 探索 | `CoinDiscoveryService.DiscoverCoins` | 向全部資訊來源收集情報、保存、彙出本輪候選幣 | Confirmed |
| 過濾 | `CoinFilteringService.FilterCoinCandidates` | 對最新成功探索的全部候選幣逐條套用過濾規則，保存過濾結果 | Confirmed |
| 洞察分析 | `CoinInsightService.AnalyzeCoinCandidates` | 對最新成功過濾保留的候選幣逐枚請 AI 產生洞察 | Confirmed |
| 獵捕裁決 | `HuntVerdictService.SynthesizeHuntVerdicts` | 對最新成功洞察中的看多洞察整輪問 CIO 一次，產出裁決並以上表的幣改寫獵捕結果表；沒有看多洞察則不問 CIO、記無資料並清空結果表 | Confirmed |
| 清空獵捕結果表 | `HuntVerdictService.ClearHuntBoard` | 獵捕回合乾淨跑完卻沒有看好的幣（某步無資料）時清空結果表 | Confirmed |
| 執行獵捕回合 | `HuntPipelineApplication.RunHuntRound` | 依序跑四個步驟並在第一個未成功的步驟停止；停在「無資料」時清空獵捕結果表 | Confirmed |
| 中斷收尾 | `PipelineRunService.FailInterruptedPipelineRuns` | 服務啟動時，把殘留「執行中」的輪次改為「失敗（被重啟中斷）」 | Confirmed |
