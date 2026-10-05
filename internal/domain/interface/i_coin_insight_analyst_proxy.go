package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_coin_insight_analyst_proxy.go -destination=mocks/mock_i_coin_insight_analyst_proxy.go -package=mocks

// ICoinInsightAnalystProxy is the AI that reads one candidate's material and answers in the agreed shape.
type ICoinInsightAnalystProxy interface {
	// AnalyzeCoin returns domains.ErrCoinInsightAnswerUnusable when the answer cannot be read; asking again may help.
	AnalyzeCoin(executionContext context.Context, material vo.CoinInsightMaterialVo) (vo.CoinInsightAnswerVo, error)
}
