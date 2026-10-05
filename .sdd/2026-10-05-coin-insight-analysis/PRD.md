# Product Requirements Document (PRD) — 候選幣 AI 洞察

**Status:** Finalized
**Version:** v1.0
**Owner:** James Hsueh
**Stakeholders:** James Hsueh（唯一使用者兼開發者）

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 過濾只回答「能不能碰」，沒有回答「該往哪個方向、有多強、為什麼」。交易者需要把散落的情報、新聞與合約市場結構濃縮成一份可比較的判斷。
- **Expected Outcome:** 每輪為每枚保留的候選幣產出一份結構化洞察（方向、強度、催化劑、風險、證據、資料缺口），可直接交給下一步的綜合裁決；AI 輸出的值一律經過正規化，不會出現範圍外的值。
- **Out of Scope:** 投資裁決、獵捕結果表、排程、AI 多輪對話或自行查資料、付費資料來源。

---

## 2. User Personas

- **Primary Role(s):** 交易者（唯一使用者兼營運者），只交易永續合約。
- **Usage Context:** 手動觸發或日後由排程在過濾之後觸發；在電腦上查閱洞察。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 為保留的候選幣產生洞察 [priority: P0]
**As a** 交易者, **I want** 每枚保留的候選幣都有一份 AI 洞察, **so that** 我能比較哪些幣值得出手。

```gherkin
Scenario: 每枚保留的幣產生一份洞察並串上過濾輪次
  Given 最新一輪成功的過濾為第 7 輪，保留 PENGU 與 STRK
  When 執行一輪洞察
  Then 產生 PENGU 與 STRK 兩份洞察
  And 洞察輪次狀態為「成功」
  And 洞察輪次記下其來源為第 7 輪過濾

Scenario: 從未有成功的過濾
  Given 從未有過成功的過濾輪次
  When 交易者手動觸發洞察
  Then 被告知「尚無成功的過濾輪次」
  And 沒有建立任何洞察輪次

Scenario: 手動觸發的洞察輪次記為手動
  When 交易者手動觸發一輪洞察
  Then 該洞察輪次的觸發來源為「手動」
```

### US-02 — 洞察內容經過正規化 [priority: P0]
**As a** 交易者, **I want** 洞察的方向與強度一定落在定義範圍, **so that** 下游比較不會被怪值干擾。

```gherkin
Scenario: 合法值照收
  Given AI 對 PENGU 回覆方向「看多」、強度 7
  When 執行一輪洞察
  Then PENGU 的洞察方向為看多、強度為 7

Scenario: 強度高於 10 夾回 10
  Given AI 對 PENGU 回覆強度 12
  When 執行一輪洞察
  Then PENGU 的洞察強度為 10

Scenario: 強度低於 1 夾回 1
  Given AI 對 PENGU 回覆強度 0
  When 執行一輪洞察
  Then PENGU 的洞察強度為 1

Scenario: 不認得的方向視為中性
  Given AI 對 PENGU 回覆方向「暴漲」
  When 執行一輪洞察
  Then PENGU 的洞察方向為中性
```

### US-03 — AI 回覆不合格的處理 [priority: P0]
**As a** 交易者, **I want** 一枚幣的 AI 問題不影響其他幣, **so that** 一輪洞察不會因單一失敗而全部作廢。

```gherkin
Scenario: 第一次不合格重問一次後成功
  Given AI 對 PENGU 第一次回覆格式不合格、第二次合格
  When 執行一輪洞察
  Then PENGU 的洞察為成功
  And AI 對 PENGU 被詢問 2 次

Scenario: 兩次都不合格即該幣分析失敗
  Given AI 對 PENGU 兩次回覆都格式不合格
  And AI 對 STRK 回覆合格
  When 執行一輪洞察
  Then PENGU 記為分析失敗，原因為「AI 回覆格式不合格」
  And STRK 的洞察為成功
  And 洞察輪次狀態為「成功」

Scenario: 全部幣分析失敗即輪次失敗
  Given AI 服務對每一枚幣都出錯
  When 執行一輪洞察
  Then 洞察輪次狀態為「失敗」
  And 失敗原因為「所有候選幣分析失敗」
```

### US-04 — 資料缺口 [priority: P1]
**As a** 交易者, **I want** 缺少素材時仍得到洞察並知道缺了什麼, **so that** 我能自行判斷這份洞察的可信度。

```gherkin
Scenario: 查不到新聞仍分析
  Given STRK 查不到近期新聞
  When 執行一輪洞察
  Then STRK 的洞察為成功
  And STRK 的資料缺口包含「查不到近期新聞」

Scenario: 查不到市場結構仍分析
  Given PONS 在幣安、Bybit、OKX 都查不到永續合約市場結構
  When 執行一輪洞察
  Then PONS 的洞察為成功
  And PONS 的資料缺口包含「查不到永續合約市場結構」
```

### US-05 — 費用上限 [priority: P1]
**As a** 交易者, **I want** 每輪 AI 分析數量有上限, **so that** 費用可預期。

```gherkin
Scenario: 超過上限只分析最早被提及的 20 枚
  Given 最新成功過濾保留 25 枚幣
  When 執行一輪洞察
  Then 只有最早被提及的 20 枚產生洞察
```

### US-06 — 查看洞察 [priority: P1]
**As a** 交易者, **I want** 查看最新洞察與某一輪的結果, **so that** 我能追溯判斷依據。

```gherkin
Scenario: 最新洞察只看最新一輪成功的洞察
  Given 第 1 輪與第 2 輪洞察都成功
  When 交易者查看最新洞察
  Then 只看到第 2 輪的洞察

Scenario: 從未成功時為空
  Given 尚無成功的洞察輪次
  When 交易者查看最新洞察
  Then 看到空清單

Scenario: 查看某一輪的結果含失敗的幣
  Given 第 2 輪洞察中 PENGU 分析失敗、STRK 成功
  When 交易者查看第 2 輪洞察的結果
  Then 看到 PENGU 為分析失敗及原因、STRK 的完整洞察

Scenario: 查看不存在的輪次
  When 交易者查看一個不存在的輪次的洞察
  Then 被告知「找不到這個輪次」
```

---

## 4. Business Flow & Logic

- **Flow:** 找最新成功過濾（無 → 拒絕）→ 取其保留的候選幣（超過 20 枚取最早被提及的 20 枚）→ 建洞察輪次（串上過濾輪次）→ 每枚幣收集素材（情報標題、新聞、市場結構、過濾結果）→ 每枚幣問 AI 一次（不合格重問一次），同時最多 3 枚 → 正規化並保存每枚幣的洞察或失敗原因 → 判定輪次狀態。
- **Core Business Rules:**
  - **訊號方向：** 看多、看空、中性三者之一；其他值一律為中性。
  - **訊號強度：** 1–10 的整數；超出夾回邊界。
  - **素材上限：** 情報標題最多 20 則（探索時間窗內，新到舊）；新聞標題最多 10 則（近 3 天）。
  - **市場結構來源順序：** 幣安 → Bybit → OKX，取第一家有該幣 USDT 永續合約的；欄位查不到的留空。
  - **資料缺口：** 新聞查不到 →「查不到近期新聞」；三家都查不到市場結構 →「查不到永續合約市場結構」；另加入 AI 自行回報的缺口。來源出錯與查無資料一視同仁記為缺口，不讓本輪失敗。
  - **重問：** AI 回覆無法解讀為規定格式（或被 AI 拒答）→ 重問一次；第二次仍不行 →「AI 回覆格式不合格」。AI 服務出錯（連線、額度、伺服器錯誤）→ 不重問，直接記為該幣分析失敗並寫明原因。
  - **輪次：** 至少一枚成功 → 成功；全部失敗 → 失敗「所有候選幣分析失敗」；輪次寫不進去即整體失敗；重啟殘留執行中 → 失敗（被重啟中斷）。
  - **AI 設定（營運者可調整）：** 模型預設 Claude Opus 5.5、思考深度預設「低」（萃取型任務）；同時分析上限 3、每輪上限 20。AI 服務位址與新聞搜尋位址可改（例如經由代理或測試）。
  - **AI 拒答備援：** 若模型以政策理由拒答，由 Claude 服務端改派備援模型在同一次詢問中代答（不額外計次）；備援也拒答才算回覆不可用。
  - **每次詢問恰好一次呼叫：** 系統不自動重送（連線、額度、伺服器錯誤都不重送），因此每輪 AI 呼叫至多「分析上限 × 2」。
  - **挑選順序：** 依探索時「最早被提及」由早到晚；查不到探索紀錄的保留幣排在最後。
  - **市場結構來源記錄：** 每份洞察記下市場結構取自哪一家交易所。
- **Edge Cases:**
  - 查看某一輪洞察時輪次編號不是正整數 → 被告知「輪次編號必須是正整數」。
  - 查看某個存在但不是洞察步驟的輪次（例如過濾輪次）的洞察 → 空清單。
  - AI 金鑰未設定 → 每枚幣都會分析失敗，輪次失敗並寫明原因。

---

## 5. UI/UX Design & Interaction

N/A — 純後端。

---

## 6. Non-Functional Requirements

- **Performance:** 每次 AI 詢問逾時 120 秒；素材收集每來源逾時 15 秒。
- **Cost:** 資料來源全部免費；AI 呼叫每輪最多 20 枚 × 2 次；固定的系統提示可被快取以降低費用。
- **Security:** AI 金鑰只從環境變數讀取，不寫入資料庫、不出現在輪次或洞察內容中。

---

## 7. Dependencies & Risks

- **External Dependencies:** Anthropic Claude API（付費，使用者金鑰）；幣安 / Bybit / OKX 公開合約行情；Google News RSS 搜尋。
- **Known Risks:** AI 判斷可能錯誤 → 洞察只是裁決的輸入，最後由裁決層綜合並附風險；新聞搜尋可能混入同名雜訊 → 以「幣名 + crypto」搜尋並交由 AI 判讀。

---

## 8. Appendix

- 需求簡報：`BRIEF.md`；上游切片：`.sdd/2026-10-05-coin-candidate-filtering/`
