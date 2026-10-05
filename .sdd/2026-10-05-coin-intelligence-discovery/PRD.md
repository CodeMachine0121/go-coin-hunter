# Product Requirements Document (PRD) — 新幣情報探索

**Status:** Finalized
**Version:** v1.0
**Owner:** James Hsueh
**Stakeholders:** James Hsueh（唯一使用者兼開發者）

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 永續合約交易者無法人工追蹤每天冒出的新幣；資訊散落在各交易所公告、新上架合約、熱門排行與鏈上新幣看板。
- **Expected Outcome:** 每一輪探索在一份清單中列出「最近 72 小時內被至少一個免費來源提到的新幣」（候選幣），並可追溯每枚候選幣來自哪些情報；任一來源故障時其餘來源照常產出，且輪次紀錄說明哪個來源壞了。
- **Out of Scope:** 候選幣過濾、AI 洞察、投資裁決、獵捕結果表、定時自動執行、推播通知、任何付費資訊來源、過舊情報的清除。

---

## 2. User Personas

- **Primary Role(s):** 交易者（唯一使用者，同時是營運者）。
- **Usage Context:** 在自己的電腦上手動觸發一輪探索或查閱結果；後續切片會改由排程自動觸發。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 從多個免費來源探索候選幣 [priority: P0]
**As a** 交易者, **I want** 一次向所有資訊來源收集情報並彙整成候選幣, **so that** 我不必逐一翻各家公告就知道最近有哪些新幣。

```gherkin
Scenario: 不同來源提到同一枚幣只成為一個候選幣
  Given 幣安上幣公告提到 CT
  And Bybit 上幣公告也提到 CT
  When 執行一輪探索
  Then 本輪候選幣只有一個 CT
  And CT 記為被 2 個來源、2 則情報提到

Scenario: 不同幣各自成為候選幣
  Given 幣安上幣公告提到 CT
  And DEX Screener 新幣看板提到 PUMP
  When 執行一輪探索
  Then 本輪候選幣為 CT 與 PUMP 兩個

Scenario: 已收過的情報不重複保存
  Given 幣安上幣公告中提到 CT 的同一則情報在上一輪已被保存
  And 該情報仍在探索時間窗內
  When 再執行一輪探索
  Then 這則情報的保存份數仍為 1
  And CT 仍為本輪候選幣
```

### US-02 — 只看時間窗內的情報 [priority: P0]
**As a** 交易者, **I want** 只有最近的情報才能產生候選幣, **so that** 已經過氣的幣不會一直出現在清單上。

```gherkin
Scenario: 時間窗內的情報產生候選幣
  Given 探索時間窗為 72 小時
  And 一則提到 CT 的情報發布於本輪開始前 10 小時
  When 執行一輪探索
  Then CT 為本輪候選幣

Scenario: 剛好落在時間窗邊界的情報仍產生候選幣
  Given 探索時間窗為 72 小時
  And 一則提到 CT 的情報發布於本輪開始前恰好 72 小時
  When 執行一輪探索
  Then CT 為本輪候選幣

Scenario: 超出時間窗的情報不產生候選幣
  Given 探索時間窗為 72 小時
  And 唯一一則提到 CT 的情報發布於本輪開始前 72 小時又 1 分鐘
  When 執行一輪探索
  Then CT 不在本輪候選幣中
```

### US-03 — 非新幣不進入候選幣 [priority: P0]
**As a** 交易者, **I want** 主流幣、穩定幣、股票類商品與無幣種的公告不會成為候選幣, **so that** 清單只剩真正的新幣。

```gherkin
Scenario: 排除幣種不成為候選幣
  Given CoinGecko 熱門排行出現 ETH
  When 執行一輪探索
  Then ETH 不在本輪候選幣中

Scenario: 傳統金融商品不成為候選幣
  Given 幣安新上架的永續合約 NKE 是股票永續合約
  When 執行一輪探索
  Then NKE 不在本輪候選幣中

Scenario: 看不出幣種的情報被保存但不產生候選幣
  Given 幣安公告「系統維護通知」沒有提到任何幣種代號
  When 執行一輪探索
  Then 這則情報被保存且沒有幣種代號
  And 本輪候選幣不因這則情報增加
```

### US-04 — 輪次紀錄與來源故障隔離 [priority: P0]
**As a** 交易者, **I want** 每一輪都留下狀態與各來源成敗, **so that** 我知道結果可不可信、哪個來源壞了。

```gherkin
Scenario: 全部來源成功且有候選幣
  Given 6 個資訊來源都成功
  And 情報中有 3 枚符合條件的幣
  When 執行一輪探索
  Then 輪次狀態為「成功」
  And 本輪候選幣為 3 個
  And 6 個來源結果皆為成功

Scenario: 部分來源失敗仍算成功
  Given OKX 上幣公告無法取得，原因為「連線逾時」
  And 其餘 5 個來源成功，情報中有 2 枚符合條件的幣
  When 執行一輪探索
  Then 輪次狀態為「成功」
  And OKX 的來源結果為失敗，原因為「連線逾時」
  And 本輪候選幣為 2 個

Scenario: 來源都成功但沒有任何候選幣
  Given 6 個資訊來源都成功
  And 所有情報都屬排除幣種或沒有幣種代號
  When 執行一輪探索
  Then 輪次狀態為「無資料」
  And 本輪候選幣為 0 個

Scenario: 全部來源失敗
  Given 6 個資訊來源都無法取得
  When 執行一輪探索
  Then 輪次狀態為「失敗」
  And 失敗原因為「所有資訊來源皆失敗」
  And 本輪沒有候選幣

Scenario: 服務重啟中斷執行中的輪次
  Given 有一輪探索的狀態為「執行中」
  When 服務重新啟動
  Then 該輪狀態變為「失敗」
  And 失敗原因為「被重啟中斷」

Scenario: 手動觸發的輪次記為手動
  When 交易者手動觸發一輪探索
  Then 該輪的觸發來源為「手動」
```

### US-05 — 查看候選幣與輪次 [priority: P1]
**As a** 交易者, **I want** 查看最新候選幣、歷次輪次與某輪的情報, **so that** 我能決定要不要深入研究某枚幣。

```gherkin
Scenario: 最新候選幣只看最新一輪成功的探索
  Given 第 1 輪探索成功，候選幣為 CT 與 PUMP
  And 第 2 輪探索成功，候選幣為 CT
  When 交易者查看最新候選幣
  Then 看到的候選幣只有 CT

Scenario: 失敗輪次不取代上一輪成功的候選幣
  Given 第 1 輪探索成功，候選幣為 CT
  And 第 2 輪探索失敗
  When 交易者查看最新候選幣
  Then 看到的候選幣只有 CT

Scenario: 從未成功過時最新候選幣為空
  Given 尚無任何成功的探索輪次
  When 交易者查看最新候選幣
  Then 看到空的候選幣清單

Scenario: 輪次歷史由新到舊
  Given 第 1 輪與第 2 輪探索都已結束
  When 交易者查看輪次歷史
  Then 第 2 輪排在第 1 輪之前

Scenario: 查看某一輪的情報
  Given 第 1 輪探索保存了 2 則情報
  When 交易者查看第 1 輪的情報
  Then 看到這 2 則情報，各自附來源、幣種代號、標題、原文連結與發布時間

Scenario: 查看不存在的輪次
  When 交易者查看一個不存在的輪次的情報
  Then 被告知「找不到這個輪次」
```

---

## 4. Business Flow & Logic

- **Flow:** 開始輪次（執行中）→ 同時向全部資訊來源索取情報 → 保存新情報（同來源同一則略過）→ 取本輪開始前 72 小時內、有幣種代號、非排除幣種、非傳統金融商品的情報 → 依幣種代號彙整成候選幣 → 依來源成敗與候選幣數決定輪次狀態。
- **Core Business Rules:**
  - **每個來源每輪最多取最新 50 則**情報。
  - **幣種代號辨識：** 合約清單以合約名稱去掉計價幣（如 CTUSDT → CT）；公告以標題中的括號代號（如「Binance Will List Cotton (CT)」→ CT）或合約名稱（「CTUSDT Perpetual」→ CT）辨識；鏈上看板與熱門排行直接採用來源給的代號。辨識不出即無幣種代號。一則公告提到多枚幣時，**每枚幣各算一則情報**。
  - 代號一律大寫比對；不同鏈上同代號視為同一枚幣。
  - **沒有發布時間的情報**（如熱門排行）以**首次被收到的時間**作為發布時間。
  - **同一則情報**的判斷：同一來源 + 同一原文識別（連結或來源給的編號）+ 同一幣種代號。
  - 探索時間窗與排除幣種名單可由營運者調整。
  - 輪次狀態：任一來源成功且候選幣 ≥1 → 成功；任一來源成功且候選幣 = 0 → 無資料；全部來源失敗 → 失敗。
  - 輪次紀錄寫不進去，該輪整體視為失敗。
- **Edge Cases:**
  - 來源回應格式不符預期 → 視為該來源失敗，原因寫明。
  - 單一來源回應慢 → 每個來源各自有逾時，不拖住整輪。
  - 服務重啟 → 殘留的執行中輪次改為失敗（被重啟中斷）。

---

## 5. UI/UX Design & Interaction

N/A — 純後端，透過 API 與 Postman 集合操作。

---

## 6. Non-Functional Requirements

- **Performance:** 一輪探索在所有來源都正常時於 30 秒內完成；每個來源逾時上限 15 秒。
- **Security:** 僅本人使用，無鑑權。
- **Cost:** 只使用免費、無需付費方案的資訊來源。
- **Analytics / Tracking:** 輪次紀錄即為追蹤資料。

---

## 7. Dependencies & Risks

- **External Dependencies:** 幣安公告、幣安永續合約清單、Bybit 公告、OKX 公告、CoinGecko 熱門排行、DEX Screener 新幣看板（皆已實測可免費取用）。
- **Known Risks:**
  - 幣安公告並非正式公開的開發者服務，格式可能無預警改變 → 來源失敗不影響其他來源。
  - 免費額度有頻率限制 → 每輪每來源只打少量請求。
  - 公告標題辨識代號有誤判可能 → 下一切片的過濾會再把關。

---

## 8. Appendix

- 需求簡報：`BRIEF.md`（同資料夾）
- 參考做法：go-stock 的多層萃取管線與管線輪次記錄
