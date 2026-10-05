# Workflow — SDD 流程與版控

## 功能切片流程

每個功能切片依序走完：

```
/clarify → /prd → /architecture → /implement（code-first）→ /improve-codebase → /contract
```

- 每個切片一個資料夾：`.sdd/{YYYY-MM-DD}-{feature-slug}/`，內含 `BRIEF.md`、`PRD.md`、`ARCH.md`、`CONTRACT.md`。
- 新功能**開新的切片資料夾**，不回頭改舊切片的 PRD。
- 業務詞彙先進 `.sdd/UL-MAP.md` 再進程式碼；程式碼與文件不可漂移。
- 流程中遇到需要決策的提問，預設採用 best practice，並把決策寫進該切片文件。
- `/improve-codebase` 的重構建議要逐條判斷，只採納**必要且符合本目錄規則**的，不照單全收。
- `/contract` 回來的缺口要視情況修正（補測試或修程式），修完再驗一次。

## 版控

- 一律在 **feature branch** 開發（`feature/{feature-slug}`），不直接在 `main` 上 commit。
- **每完成一個段落（一個實作步驟、一份文件、一次 App 組裝）就 commit 一次**，不累積成大 commit。
- commit message 遵循 **Conventional Commits**，**一律英文**（`feat(scope): ...`、`docs: ...`、`refactor: ...`、`test: ...`）。
- commit message、分支名、PR 標題**不得出現任何 SDD 代號**（測試案例編號、AC 編號、文件名如 PRD/ARCH）。描述行為，不描述流程。
- commit 前自行確認 `go build ./...`、`go vet ./...`、`make test` 全過。
