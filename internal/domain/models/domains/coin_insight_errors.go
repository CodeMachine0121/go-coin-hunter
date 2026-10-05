package domains

import "errors"

var (
	ErrNoSucceededFilteringRun = errors.New("尚無成功的過濾輪次")
	// ErrCoinInsightAnswerUnusable means the analyst answered, but not in a form that can be read; asking again may help.
	ErrCoinInsightAnswerUnusable = errors.New("AI 回覆格式不合格")
)

const AllCoinAnalysesFailedReason = "所有候選幣分析失敗"
