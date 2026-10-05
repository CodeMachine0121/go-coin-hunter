# Product Requirements Document (PRD) — 獵捕裁決與獵捕結果表

**Status:** Finalized
**Version:** v1.0
**Owner:** James Hsueh
**Stakeholders:** James Hsueh（唯一使用者兼開發者）

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 逐幣洞察彼此獨立，沒有人綜合比較、分配部位、設定停損；交易者需要一張「現在該看哪些幣、怎麼做」的單一清單。
- **Expected Outcome:** 每輪成功裁決後，獵捕結果表恰好反映本輪的全部裁決（本輪有的覆蓋、沒有的移除），每列都有可執行的操作、風險參數與理由；所有數值都在安全範圍內。
- **Out of Scope:** 下單、推播、排程、績效回測。

---

## 2. User Personas

- **Primary Role(s):** 交易者（唯一使用者），只做永續合約。
- **Usage Context:** 手動觸發或日後由排程在洞察之後觸發；打開獵捕結果表決定當天操作。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 改寫獵捕結果表 [priority: P0]
**As a** 交易者, **I want** 結果表永遠只反映最新一輪成功的裁決, **so that** 我看到的清單不會混著過期的幣。

```gherkin
Scenario: 空表寫入本輪裁決
  Given 獵捕結果表為空
  And 本輪裁決的幣為 BTC、ETH、BNB
  When 執行一輪裁決
  Then 獵捕結果表有 BTC、ETH、BNB 三列
  And 三列的計算時間皆為本輪裁決時間

Scenario: 本輪沒有的幣從表上移除
  Given 獵捕結果表有 BTC、ETH、BNB
  And 本輪裁決的幣為 BTC、ETH
  When 執行一輪裁決
  Then 獵捕結果表只有 BTC、ETH 兩列
  And BTC、ETH 為本輪的裁決內容與計算時間

Scenario: 失敗的裁決不改動結果表
  Given 獵捕結果表有 BTC
  And CIO 兩次回覆都格式不合格
  When 執行一輪裁決
  Then 裁決輪次狀態為「失敗」
  And 失敗原因為「AI 回覆格式不合格」
  And 獵捕結果表仍只有原本的 BTC
```

### US-02 — 裁決數值正規化 [priority: P0]
**As a** 交易者, **I want** 槓桿、部位、信心永遠在安全範圍, **so that** AI 的誇張建議不會直接變成風險。

```gherkin
Scenario: 槓桿與部位超出上限被夾回
  Given CIO 對 PENGU 給出做多、槓桿 20 倍、部位 30%
  When 執行一輪裁決
  Then PENGU 的建議槓桿為 5 倍、部位大小為 10%

Scenario: 不認得的操作視為觀望
  Given CIO 對 PENGU 給出操作「梭哈」
  When 執行一輪裁決
  Then PENGU 的操作為觀望
  And 建議槓桿為 0、部位大小為 0%

Scenario: 信心超出 100 被夾回
  Given CIO 對 PENGU 給出信心 130
  When 執行一輪裁決
  Then PENGU 的信心為 100
```

### US-03 — 停損停利價 [priority: P0]
**As a** 交易者, **I want** 停損停利以價格呈現且一定在正確的一側, **so that** 我可以直接掛單。

```gherkin
Scenario: 做多的停損在下、停利在上
  Given PENGU 最新價格為 0.01
  And CIO 對 PENGU 給出做多、停損距離 10%、停利距離 30%
  When 執行一輪裁決
  Then PENGU 的停損價為 0.009、停利價為 0.013

Scenario: 做空的停損在上、停利在下
  Given PENGU 最新價格為 0.01
  And CIO 對 PENGU 給出做空、停損距離 10%、停利距離 30%
  When 執行一輪裁決
  Then PENGU 的停損價為 0.011、停利價為 0.007

Scenario: 停損距離過小被夾到 1%
  Given PENGU 最新價格為 0.01
  And CIO 對 PENGU 給出做多、停損距離 0.5%
  When 執行一輪裁決
  Then PENGU 的停損距離為 1%、停損價為 0.0099

Scenario: 查不到最新價格的做多改判觀望
  Given PONS 查不到最新價格
  And CIO 對 PONS 給出做多
  When 執行一輪裁決
  Then PONS 的操作為觀望
  And 理由為「查不到最新價格，無法設定停損」
```

### US-04 — 洞察與裁決的對應 [priority: P0]
**As a** 交易者, **I want** 每一枚有洞察的幣都有裁決、不多也不少, **so that** 結果表完整且可信。

```gherkin
Scenario: CIO 漏掉的幣記為觀望
  Given 洞察有 PENGU 與 STRK
  And CIO 只對 PENGU 給出裁決
  When 執行一輪裁決
  Then STRK 的操作為觀望
  And 理由為「CIO 未給出裁決」

Scenario: CIO 多給的幣被忽略
  Given 洞察只有 PENGU
  And CIO 另外對 DOGE 給出裁決
  When 執行一輪裁決
  Then 獵捕結果表沒有 DOGE

Scenario: 從未有成功的洞察
  Given 從未有過成功的洞察輪次
  When 交易者手動觸發裁決
  Then 被告知「尚無成功的洞察輪次」
  And 沒有建立任何裁決輪次
```

### US-05 — 輪次與查詢 [priority: P1]
**As a** 交易者, **I want** 追溯每輪裁決並查看結果表, **so that** 我能檢討判斷。

```gherkin
Scenario: 成功的裁決輪次串上洞察輪次
  Given 最新一輪成功的洞察為第 9 輪
  When 執行一輪裁決
  Then 裁決輪次狀態為「成功」
  And 裁決輪次記下其來源為第 9 輪洞察
  And 觸發來源為「手動」

Scenario: 結果表依信心由高到低
  Given 獵捕結果表有信心 40 的 STRK 與信心 80 的 PENGU
  When 交易者查看獵捕結果表
  Then PENGU 排在 STRK 之前

Scenario: 查看某一輪的裁決
  Given 第 3 輪裁決了 PENGU 與 STRK
  When 交易者查看第 3 輪的裁決
  Then 看到 PENGU 與 STRK 的裁決

Scenario: 查看不存在的輪次
  When 交易者查看一個不存在的輪次的裁決
  Then 被告知「找不到這個輪次」
```

---

## 4. Business Flow & Logic

- **Flow:** 找最新成功洞察（無 → 拒絕）→ 取其分析成功的洞察 → 建裁決輪次（串上洞察輪次）→ 為每枚幣重新取市場結構 → 整輪問 CIO 一次（不可用重問一次）→ 正規化每枚幣的裁決 → 保存本輪裁決紀錄 → 以一筆交易改寫獵捕結果表 → 輪次成功。
- **Core Business Rules:**
  - **操作：** 做多 `long`、做空 `short`、觀望 `watch`、避開 `avoid`；其他一律觀望。
  - **數值範圍：** 信心 0–100；槓桿 1–5（做多 / 做空），觀望 / 避開為 0；部位 0%–10%，觀望 / 避開為 0；停損距離 1%–50%；停利距離 1%–200%，**做空的停利距離上限 90%**（價格最多跌到零，停利價必須大於零）；皆夾回邊界。信心與槓桿若給小數，先四捨五入再夾。
  - **停損停利價：** 以裁決當下的最新價格換算：做多停損價 = 價 ×（1 − 停損距離）、停利價 = 價 ×（1 + 停利距離）；做空相反。金額以精確小數計算。觀望 / 避開不設停損停利。
  - **查不到最新價格：** 最新價格不存在、或不大於 0 → 做多 / 做空改判觀望（槓桿、部位歸 0），理由「查不到最新價格，無法設定停損」。
  - **對應：** 洞察中每一枚分析成功的幣恰好一筆裁決；CIO 漏的 → 觀望「CIO 未給出裁決」；多給的 → 忽略；同一枚幣給了多筆 → 採第一筆。代號比對不分大小寫。任一筆裁決缺少欄位即視為整份回覆格式不合格（觸發重問）。
  - **參考價格：** 每筆做多 / 做空裁決記下換算停損停利所用的最新價格（參考價格）。
  - **重問：** CIO 回覆不可用（格式不合、拒答、截斷）重問一次；仍不可用 → 輪次失敗「AI 回覆格式不合格」；AI 服務出錯不重問，輪次失敗並寫明原因。每次詢問恰好一次呼叫，伺服器端拒答備援同洞察切片。
  - **改寫結果表：** 先保存本輪裁決紀錄（輪次歷史），再以**一筆交易**改寫結果表：本輪有的幣覆蓋（無則新增）並以本輪時間為計算時間、本輪沒有的幣刪除。改寫失敗 → 輪次失敗、結果表不變（已保存的裁決紀錄留作該失敗輪次的歷史）。
  - **失敗的回應：** AI 無法給出可用回覆 → 手動觸發仍得到該失敗輪次；裁決紀錄或結果表寫不進去 → 輪次標失敗，手動觸發得到伺服器錯誤。
  - **結果表排序：** 依信心由高到低，同信心依幣種代號。
  - **AI 設定（可調整）：** 模型預設 Claude Opus 5.5、思考深度預設「高」（綜合判斷）；單次詢問逾時 180 秒；市場結構每來源逾時 15 秒，查不到即無最新價格。
  - **輪次：** 重啟殘留執行中 → 失敗（被重啟中斷）。
- **Edge Cases:**
  - 查看某一輪裁決時輪次編號不是正整數 → 「輪次編號必須是正整數」；存在但非裁決步驟的輪次 → 空清單。

---

## 5. UI/UX Design & Interaction

N/A — 純後端。

---

## 6. Non-Functional Requirements

- **Cost:** 每輪 CIO 至多 2 次呼叫；市場結構資料免費。
- **Consistency:** 獵捕結果表不會出現「一半新一半舊」的狀態。
- **Security:** AI 金鑰只從環境變數讀取，不出現在任何紀錄中。

---

## 7. Dependencies & Risks

- **External Dependencies:** Anthropic Claude API（付費）；幣安 / Bybit / OKX 公開行情。
- **Known Risks:** AI 建議可能錯誤 → 數值一律夾在保守範圍、停損由系統換算、不下單；裁決僅供決策參考。

---

## 8. Appendix

- 需求簡報：`BRIEF.md`；上游切片：`.sdd/2026-10-05-coin-insight-analysis/`
