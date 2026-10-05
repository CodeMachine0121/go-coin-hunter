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
| ORM / DB | GORM + PostgreSQL，Code First、啟動時 `AutoMigrate` |
| AI | Anthropic Claude（`anthropic-sdk-go`） |
| 金額 | `shopspring/decimal` |
| 測試 | `testing` + `testify` + `go.uber.org/mock`（`go tool mockgen`） |
| 資料來源 | **只用免費來源或免費額度**，不接任何付費 API |

## 啟動

```
cp .env.example .env   # 所有變數皆有預設值，可省略
# 需要一個 PostgreSQL（預設 localhost:5432、postgres/postgres、資料庫 go_coin_hunter），例如：
#   docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16-alpine
#   docker exec postgres psql -U postgres -c 'CREATE DATABASE go_coin_hunter' -c 'CREATE DATABASE go_coin_hunter_test'
go mod download
make start             # curl localhost:8080/health → {"status":"Healthy"}
make test              # go test ./... -race（未設 TEST_POSTGRES_DSN 時儲存層測試會跳過）
TEST_POSTGRES_DSN="host=localhost port=5432 user=postgres password=postgres dbname=go_coin_hunter_test sslmode=disable" make test-storage
make mock              # 重新產生 mock
```

## 環境變數

| 變數 | 預設值 | 用途 |
| :--- | :--- | :--- |
| `SERVER_ADDRESS` | `:8080` | HTTP 監聽位址 |
| `POSTGRES_HOST` / `POSTGRES_PORT` | `localhost` / `5432` | PostgreSQL 位址 |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` | `postgres` / `postgres` | PostgreSQL 帳密 |
| `POSTGRES_DATABASE` | `go_coin_hunter` | 資料庫名稱 |
| `POSTGRES_SSL_MODE` | `disable` | SSL 模式 |
| `TEST_POSTGRES_DSN` | 空（儲存層測試跳過） | 測試用資料庫連線字串，資料庫名稱**必須**以 `_test` 結尾（測試會清空資料表） |
| `BACKGROUND_JOBS_ENABLED` | `true` | 背景 job 總開關 |
| `DISCOVERY_WINDOW_HOURS` | `72` | 探索時間窗：只有這段時間內發布的情報才產生候選幣 |
| `DISCOVERY_EXCLUDED_COIN_SYMBOLS` | `BTC,ETH,BNB,SOL,XRP,USDT,USDC,FDUSD,DAI,TUSD,USDE` | 排除幣種（主流幣、穩定幣），逗號分隔 |
| `BINANCE_WEB_BASE_URL` / `BINANCE_FUTURES_BASE_URL` / `BYBIT_BASE_URL` / `OKX_BASE_URL` / `COINGECKO_BASE_URL` / `DEXSCREENER_BASE_URL` | 各官方網址 | 資訊來源網址（測試或代理時覆寫） |
| `FILTER_MAXIMUM_TAX_RATE` | `0.1` | 安全檢查：買賣稅上限（比率，含上限通過） |
| `FILTER_MINIMUM_DAILY_VOLUME_USD` | `1000000` | 流動性門檻：24 小時成交額下限（美元，含） |
| `FILTER_MINIMUM_FDV_USD` / `FILTER_MAXIMUM_FDV_USD` | `10000000` / `1000000000` | 完全稀釋估值區間（美元，含兩端） |
| `FILTER_MINIMUM_CIRCULATING_RATIO` | `0.2` | 流通比下限（含） |
| `FILTER_UNLOCK_LOOKAHEAD_DAYS` | `14` | 解鎖時程觀察天數 |
| `FILTER_MAXIMUM_UNLOCK_RATIO` | `0.05` | 觀察期內累計解鎖量占流通量的上限（達到即淘汰） |
| `GOPLUS_BASE_URL` / `DEFILLAMA_DATASETS_BASE_URL` | 各官方網址 | 過濾資料來源網址 |

## 資訊來源（全部免費、免金鑰）

| 來源 | 內容 |
| :--- | :--- |
| 幣安上幣公告 | New Cryptocurrency Listing 公告，從標題辨識代號 |
| 幣安永續合約 | 最新上架的永續合約；股票等傳統金融合約標為非新幣 |
| Bybit 上幣公告 | New Listings 公告 |
| OKX 上幣公告 | New Listings 公告 |
| CoinGecko 熱門 | 熱門幣排行（無發布時間，以首次收到時間為準） |
| DEX Screener | 最新 token profile，再查代號與最早交易對建立時間 |

新增來源：實作 `IInformationSourceProxy`（`internal/domain/interface/i_information_source_proxy.go`），在 `cmd/server/dependencies.go` 的 `informationSourcesFor` 多加一行即可。

## 過濾規則（策略模式）

每條規則一個 `XxxFilterHandler`（`internal/domain/handler/`），實作 `ICoinCandidateFilterHandler`，只讀幣種檔案、不碰外部來源。
新增規則：寫一個 handler，在 `cmd/server/dependencies.go` 的 `filterHandlersFor` 多一行。

| 規則 | 資料來源（免費） |
| :--- | :--- |
| 安全檢查 | GoPlus（每次一個合約，每 2 秒一次；只查已上永續合約的幣） |
| 流動性門檻、完全稀釋估值、流通比 | CoinGecko（優先）、DEX Screener（鏈上幣備援，無供給量） |
| 解鎖時程 | DefiLlama 公開 emissions 資料集 |
| 是否已上永續合約 | 幣安、Bybit、OKX 永續合約清單（只算加密原生 USDT 永續） |

## API Routes

- `GET /health`
- `POST /coin-discoveries` — 手動觸發一輪探索，回傳該輪輪次（含各來源成敗）
- `GET /coin-candidates/latest` — 最新一輪**成功**探索的候選幣（從未成功過為空陣列）
- `POST /coin-filterings` — 手動觸發一輪過濾（對最新成功探索的候選幣）；從未有成功探索時回 409「尚無成功的探索輪次」
- `GET /coin-filter-results/latest-kept` — 最新一輪成功過濾後**保留**的候選幣，附六條規則的結果與理由
- `GET /pipeline-runs/:pipelineRunId/coin-filter-results` — 某一輪過濾的全部結果（含淘汰的幣；非正整數 400、查無輪次 404）
- `GET /pipeline-runs` — 管線輪次歷史，新到舊，含各來源結果
- `GET /pipeline-runs/:pipelineRunId/coin-intelligences` — 某一輪首次保存的情報（非正整數 400、查無輪次 404）

Postman 測試集在 `postman/`，新增 / 修改路由需同步更新。
