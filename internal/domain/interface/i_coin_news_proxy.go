package _interface

import (
	"context"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_coin_news_proxy.go -destination=mocks/mock_i_coin_news_proxy.go -package=mocks

type ICoinNewsProxy interface {
	// FindRecentHeadlines returns at most limit headlines published since the given time, newest first.
	FindRecentHeadlines(executionContext context.Context, coinSymbol string, since time.Time, limit int) ([]vo.HeadlineVo, error)
}
