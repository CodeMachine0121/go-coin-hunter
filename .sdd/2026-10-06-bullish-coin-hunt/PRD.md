# Product Requirements Document (PRD) — 只獵看好的幣

**Status:** Finalized
**Version:** v1.1（程式碼審查後：資金費率改以 8 小時折算比較、市場結構查詢限流並有獨立預算、門檻防呆）
**Owner:** James Hsueh
**Stakeholders:** James Hsueh（唯一使用者兼開發者）

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 專案宗旨是找出值得做多的新幣，但目前管線對每一枚候選幣都送進 AI、都給出做多 / 做空 / 觀望 / 避開，獵捕結果表上混著大量觀望與避開，交易者得自己翻找；明顯在下跌、資金撤離或多方過熱的幣也一路花掉 AI 成本才被判為不看好。
- **Expected Outcome:**
  - 明顯不利做多的候選幣在 AI 之前就被規則淘汰，並附理由。
  - 只有看多且訊號夠強的洞察會交給 CIO；沒有時不問 CIO。
  - 獵捕結果表上的每一列都是「做多、且信心達門檻」的建議；沒有任何看好的幣時結果表是空的，而不是留著過時的建議。
  - 每輪交給 CIO 的全部裁決仍完整保存於歷史。
- **Out of Scope:** 調整資訊來源；調整洞察分析師的中立判斷方式；不看好的幣的專屬結果表或管線；單獨手動執行探索或過濾得到「無資料」時清空結果表；下單、推播、回測。

---

## 2. User Personas

- **Primary Role(s):** 交易者（唯一使用者），只做永續合約，只找做多機會。
- **Usage Context:** 排程每 4 小時自動執行一個獵捕回合，或手動一鍵觸發；交易者打開獵捕結果表，直接看到本輪值得做多的幣。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 動能規則在 AI 之前淘汰不利做多的幣 [priority: P0]
**As a** 交易者, **I want** 明顯下跌、已大漲、資金撤離或多方過熱的候選幣在過濾時就被淘汰, **so that** AI 只花在有機會做多的幣上。

三條規則都看候選幣**此刻**的永續合約市場結構（依幣安 → Bybit → OKX 取第一家有的）。查不到時記「無資料」，不淘汰。

```gherkin
Scenario: 24 小時漲幅溫和，通過
  Given PENGU 24 小時漲跌幅為 +12%
  When 執行一輪過濾
  Then PENGU 的「24 小時漲跌幅」規則結果為通過

Scenario: 跌幅剛好 10%，通過
  Given PENGU 24 小時漲跌幅為 -10%
  When 執行一輪過濾
  Then PENGU 的「24 小時漲跌幅」規則結果為通過

Scenario: 跌幅超過 10%，淘汰
  Given PENGU 24 小時漲跌幅為 -10.5%
  When 執行一輪過濾
  Then PENGU 的「24 小時漲跌幅」規則結果為淘汰
  And 理由寫明 24 小時跌幅 10.5% 超過下限 10%
  And PENGU 不被保留

Scenario: 漲幅剛好 60%，通過
  Given PENGU 24 小時漲跌幅為 +60%
  When 執行一輪過濾
  Then PENGU 的「24 小時漲跌幅」規則結果為通過

Scenario: 漲幅超過 60%，追高風險淘汰
  Given PENGU 24 小時漲跌幅為 +61%
  When 執行一輪過濾
  Then PENGU 的「24 小時漲跌幅」規則結果為淘汰
  And 理由寫明 24 小時漲幅 61% 超過上限 60%

Scenario: 查不到市場結構，三條動能規則皆無資料
  Given 沒有任何交易所回報 PENGU 的永續合約市場結構
  When 執行一輪過濾
  Then PENGU 的三條動能規則結果皆為無資料，理由為「查不到永續合約市場結構」
  And PENGU 不因動能規則被淘汰

Scenario: 持倉量增加，通過
  Given PENGU 持倉量 24 小時變化為 +25%
  When 執行一輪過濾
  Then PENGU 的「持倉量變化」規則結果為通過

Scenario: 持倉量剛好減少 10%，通過
  Given PENGU 持倉量 24 小時變化為 -10%
  When 執行一輪過濾
  Then PENGU 的「持倉量變化」規則結果為通過

Scenario: 持倉量減少超過 10%，淘汰
  Given PENGU 持倉量 24 小時變化為 -11%
  When 執行一輪過濾
  Then PENGU 的「持倉量變化」規則結果為淘汰
  And 理由寫明持倉量 24 小時減少 11% 超過下限 10%

Scenario: 交易所沒提供持倉量變化
  Given PENGU 有市場結構但沒有持倉量 24 小時變化
  When 執行一輪過濾
  Then PENGU 的「持倉量變化」規則結果為無資料，理由為「查不到持倉量 24 小時變化」

Scenario: 資金費率正常，通過
  Given PENGU 每 8 小時結算一次，每期資金費率為 0.01%
  When 執行一輪過濾
  Then PENGU 的「資金費率過熱」規則結果為通過

Scenario: 資金費率剛好 0.1%，通過
  Given PENGU 每 8 小時結算一次，每期資金費率為 0.1%
  When 執行一輪過濾
  Then PENGU 的「資金費率過熱」規則結果為通過

Scenario: 資金費率超過 0.1%，多方過熱淘汰
  Given PENGU 每 8 小時結算一次，每期資金費率為 0.11%
  When 執行一輪過濾
  Then PENGU 的「資金費率過熱」規則結果為淘汰
  And 理由寫明資金費率 0.11% 高於上限 0.1%

Scenario: 負資金費率不淘汰
  Given PENGU 每期資金費率為 -0.3%
  When 執行一輪過濾
  Then PENGU 的「資金費率過熱」規則結果為通過

Scenario: 每小時結算的費率先折合成 8 小時再比較
  Given PENGU 每 1 小時結算一次，每期資金費率為 0.08%
  When 執行一輪過濾
  Then PENGU 的「資金費率過熱」規則結果為淘汰
  And 理由寫明資金費率每 1 小時 0.08%，折合每 8 小時 0.64%，高於上限 0.1%

Scenario: 每 4 小時結算、折合後剛好在上限，通過
  Given PENGU 每 4 小時結算一次，每期資金費率為 0.05%
  When 執行一輪過濾
  Then PENGU 的「資金費率過熱」規則結果為通過

Scenario: 交易所沒提供資金費率
  Given PENGU 有市場結構但沒有資金費率
  When 執行一輪過濾
  Then PENGU 的「資金費率過熱」規則結果為無資料，理由為「查不到資金費率」
```

### US-02 — 只把看多的洞察交給 CIO [priority: P0]
**As a** 交易者, **I want** CIO 只看看多且訊號夠強的幣, **so that** 裁決成本與注意力集中在做多機會上。

```gherkin
Scenario: 只有看多的幣交給 CIO
  Given 最新一輪成功洞察中 PENGU 看多強度 8、STRK 看空強度 9
  When 執行一輪裁決
  Then 交給 CIO 的幣只有 PENGU
  And 本輪裁決只有 PENGU 一筆

Scenario: 看多強度剛好 6，交給 CIO
  Given 最新一輪成功洞察中 PENGU 看多強度 6
  When 執行一輪裁決
  Then PENGU 交給 CIO

Scenario: 看多強度 5，不交給 CIO
  Given 最新一輪成功洞察中 PENGU 看多強度 5、STRK 看多強度 7
  When 執行一輪裁決
  Then 交給 CIO 的幣只有 STRK

Scenario: 沒有看多的洞察，不問 CIO
  Given 獵捕結果表有 ARB
  And 最新一輪成功洞察中 PENGU 中性強度 9、STRK 分析失敗
  When 執行一輪裁決
  Then 不詢問 CIO
  And 裁決輪次狀態為「無資料」
  And 本輪沒有任何裁決
  And 獵捕結果表為空
```

### US-03 — CIO 只在做多與不做之間取捨 [priority: P0]
**As a** 交易者, **I want** CIO 不再給做空建議, **so that** 結果完全對齊「找做多機會」。

```gherkin
Scenario: 做多附完整風險參數
  Given PENGU 最新價格為 0.01
  And CIO 對 PENGU 給出做多、停損 10%、停利 30%
  When 執行一輪裁決
  Then PENGU 的操作為做多
  And 停損價為 0.009、停利價為 0.013

Scenario: CIO 給做空視為觀望
  Given CIO 對 PENGU 給出做空、槓桿 3 倍、部位 5%
  When 執行一輪裁決
  Then PENGU 的操作為觀望
  And 建議槓桿為 0、部位大小為 0%、沒有停損價與停利價

Scenario: 避開保存於歷史但不上表
  Given CIO 對 PENGU 給出避開
  When 執行一輪裁決
  Then 本輪裁決中 PENGU 的操作為避開
  And 獵捕結果表沒有 PENGU
```

### US-04 — 獵捕結果表只放做多且信心達門檻的幣 [priority: P0]
**As a** 交易者, **I want** 結果表上每一列都是值得做多的建議, **so that** 我不用再自己翻找。

```gherkin
Scenario: 只有做多上表，全部裁決留在歷史
  Given CIO 對 PENGU 給出做多信心 72、STRK 觀望、ARB 避開
  When 執行一輪裁決
  Then 獵捕結果表只有 PENGU
  And 本輪裁決有 PENGU、STRK、ARB 三筆

Scenario: 做多信心剛好 50，上表
  Given CIO 對 PENGU 給出做多信心 50
  When 執行一輪裁決
  Then 獵捕結果表有 PENGU

Scenario: 做多信心 49，不上表
  Given CIO 對 PENGU 給出做多信心 49
  When 執行一輪裁決
  Then 獵捕結果表沒有 PENGU
  And 本輪裁決中 PENGU 為做多信心 49

Scenario: CIO 全部不做多，結果表清空
  Given 獵捕結果表有 PENGU
  And CIO 對本輪每一枚幣都給出觀望或避開
  When 執行一輪裁決
  Then 裁決輪次狀態為「成功」
  And 獵捕結果表為空

Scenario: 本輪上表的幣取代舊表
  Given 獵捕結果表有 PENGU、STRK
  And 本輪做多且信心達門檻的是 STRK、ARB
  When 執行一輪裁決
  Then 獵捕結果表只有 STRK、ARB
  And STRK 為本輪的裁決內容與計算時間
```

### US-05 — 回合乾淨跑完但沒有看好的幣時清空結果表 [priority: P0]
**As a** 交易者, **I want** 回合乾淨跑完卻沒有看好的幣時結果表是空的、失敗時則保留原表, **so that** 我不會照著過時的做多建議下單，也不會因為一次故障就失去清單。

```gherkin
Scenario: 過濾全部淘汰，結果表清空
  Given 獵捕結果表有 PENGU
  And 本輪過濾把所有候選幣都淘汰
  When 執行一個獵捕回合
  Then 回合停在過濾，原因為「過濾未成功：noData」（沿用既有回合原因格式「{步驟}未成功：{狀態}」）
  And 獵捕結果表為空

Scenario: 探索沒有候選幣，結果表清空
  Given 獵捕結果表有 PENGU
  And 本輪探索在時間窗內沒有任何候選幣
  When 執行一個獵捕回合
  Then 回合停在探索
  And 獵捕結果表為空

Scenario: 洞察失敗，結果表保留
  Given 獵捕結果表有 PENGU
  And 本輪每一枚幣的洞察分析都失敗
  When 執行一個獵捕回合
  Then 回合停在洞察
  And 獵捕結果表仍只有 PENGU

Scenario: 裁決失敗，結果表保留
  Given 獵捕結果表有 PENGU
  And CIO 兩次回覆都格式不合格
  When 執行一個獵捕回合
  Then 回合停在裁決
  And 獵捕結果表仍只有 PENGU
```

---

## 4. Business Flow & Logic

- **Flow：** 探索（不變）→ 過濾（既有六條規則 ＋ 三條動能規則）→ 洞察（不變，分析師仍中立判斷）→ 裁決（只收看多且強度 ≥ 6 的洞察；沒有則「無資料」不問 CIO）→ 改寫獵捕結果表（只放做多且信心 ≥ 50）。
- **Core Business Rules：**
  - 動能規則門檻（皆含邊界、皆可調整）：24 小時漲跌幅 −10% ～ +60%；持倉量 24 小時變化 ≥ −10%；折合每 8 小時的資金費率 ≤ 0.1%（交易所每 8、4 或 1 小時結算一次，先依結算週期折算再比較；查不到週期視為 8 小時）。
  - 門檻防呆：訊號強度門檻夾在 1–10、信心門檻夾在 0–100；漲跌幅下限高於上限時兩者一起退回預設值。
  - 動能規則所需的市場結構於過濾時取得，與其他資料來源並行、有自己的時間預算，不吃掉其他來源的時間；某一家交易所失敗就換下一家，全部查不到記「無資料」，不讓整輪過濾失敗（市場結構是逐幣的輔助資料，不像市值或合約清單那樣是整輪必備）。
  - 看多洞察門檻：方向看多、強度 ≥ 6（可調整），且分析成功。
  - CIO 操作只有做多、觀望、避開；做空與其他值一律觀望。做多的建議槓桿、部位、停損停利範圍沿用原裁決規則；原本只為做空設的停利上限不再需要。
  - 結果表門檻：做多且信心 ≥ 50（可調整）。
  - 結果表改寫：依「本輪上表的幣」整批改寫，全有或全無；沒有任何幣上表即清空。
  - 裁決輪次狀態：沒有看多洞察 →「無資料」（不問 CIO）；CIO 回覆可用且結果表改寫成功 →「成功」（即使沒有幣上表）；否則「失敗」。
  - 獵捕回合：任一步以「無資料」結束 → 回合停在該步，結果表清空；任一步「失敗」→ 回合停在該步，結果表不動。
- **Edge Cases：**
  - 結果表清空本身失敗：回合照樣停在原步驟，並在回合原因中附註清空失敗，結果表維持原樣（下一輪會再改寫）。
  - 既有結果表上殘留的做空 / 觀望 / 避開列：下一次改寫時依新規則移除。
  - 歷史中既有的做空裁決保留不動。

---

## 5. UI/UX Design & Interaction

- N/A（無介面；交易者透過查詢獵捕結果表與輪次歷史取得結果，回傳形狀不變）。
- 空狀態：結果表為空即代表最近一個乾淨跑完的回合沒有看好的幣；搭配輪次歷史可分辨是「沒有看好的幣」還是「回合失敗」。

---

## 6. Non-Functional Requirements

- **Performance：** 過濾時逐幣查詢市場結構需同時進行，不得讓過濾時間隨候選幣數線性拉長到失控；每家交易所每次查詢沿用既有逾時。
- **Rate limits：** 所有步驟共用同一個市場結構查詢，同時查詢的幣數有上限，避免一次大量候選幣觸發交易所的免費請求限制。
- **Cost：** 沒有看多洞察時不呼叫 CIO；只用免費資料來源。
- **Security / Compatibility：** N/A。

---

## 7. Dependencies & Risks

- **External Dependencies：** 幣安、Bybit、OKX 的公開永續合約行情（洞察階段已在使用）。
- **Known Risks：**
  - 門檻過緊會讓結果表長期為空 → 所有門檻可調整，並可從過濾結果與歷史裁決回頭校準。
  - 新幣上架初期行情波動大，24 小時漲跌幅上限 60% 可能擋掉部分強勢幣 → 屬刻意的追高保護，可調整。

---

## 8. Appendix

- 需求簡報：`BRIEF.md`（同資料夾）。
- 相關切片：`2026-10-05-coin-candidate-filtering`、`2026-10-05-coin-hunt-verdict`、`2026-10-05-coin-hunt-scheduled-pipeline`。
