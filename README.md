# go-coin-hunter

加密貨幣新幣獵捕引擎（**純 Go 後端**，個人側項目，非公開對外服務）。

自動從免費資訊來源探索新上線 / 即將上線的幣，先用規則式過濾淘汰雜訊與高風險標的，再交給 AI 萃取訊號，
最後由一個「CIO（投資長）」人格的 AI 裁決出可執行的操作建議，彙整成一張**每輪覆寫**的獵捕結果表。
不代操、不下單，純資訊 / 決策輔助；面向**永續合約**交易者，所以「是否已上永續合約」是一級公民。

資訊萃取流程沿用 go-stock 的多層管線：

```
① 探索      IInformationProvider 清單（每個資訊來源一個 Provider）→ 情報 + 候選幣
② 過濾      ICoinCandidateFilter 清單（策略模式，每條規則一個 Handler）→ 淘汰 / 保留
③ 洞察      Claude 依情報與市場結構萃取每檔候選幣的訊號（最多 1–2 輪 AI 互動）
④ 裁決      CIO 綜合洞察 → 做多 / 做空 / 觀察 / 避開，覆寫獵捕結果表
⑤ 排程      背景 job 依序串起 ①→④，每步落地一筆管線輪次記錄
```

切片規劃與進度見 [.sdd/ROADMAP.md](.sdd/ROADMAP.md)；業務詞彙見 [.sdd/UL-MAP.md](.sdd/UL-MAP.md)；
實作規範見 [CLAUDE.md](CLAUDE.md)。

## Tech Stack

| 層面 | 選型 |
| :--- | :--- |
| 語言 | Go 1.26 |
| Web 框架 | Gin |
| ORM / DB | GORM + SQLite（`glebarez/sqlite`，純 Go、無 CGO），Code First、`AutoMigrate` |
| AI | Anthropic Claude（`anthropic-sdk-go`） |
| 金額 | `shopspring/decimal` |
| 測試 | `testing` + `testify` + `go.uber.org/mock`（`go tool mockgen`） |
| 資料來源 | **只用免費來源或免費額度**，不接任何付費 API |

## 啟動

```
cp .env.example .env   # 所有變數皆有預設值，可省略
go mod download
make start             # curl localhost:8080/health → {"status":"Healthy"}
make test              # go test ./... -race
make mock              # 重新產生 mock
```

## 環境變數

| 變數 | 預設值 | 用途 |
| :--- | :--- | :--- |
| `SERVER_ADDRESS` | `:8080` | HTTP 監聽位址 |
| `SQLITE_DB_PATH` | `./data/go-coin-hunter.sqlite3` | SQLite 檔案路徑 |
| `BACKGROUND_JOBS_ENABLED` | `true` | 背景 job 總開關 |

## API Routes

- `GET /health`

Postman 測試集在 `postman/`，新增 / 修改路由需同步更新。
