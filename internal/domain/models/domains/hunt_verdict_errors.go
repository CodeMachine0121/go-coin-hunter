package domains

import "errors"

var (
	ErrNoSucceededInsightRun = errors.New("尚無成功的洞察輪次")
	// ErrHuntVerdictAnswerUnusable means the strategist answered, but not in a form that can be read; asking again may help.
	ErrHuntVerdictAnswerUnusable = errors.New("AI 回覆格式不合格")
)
