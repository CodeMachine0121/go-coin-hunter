package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_token_unlock_schedule_proxy.go -destination=mocks/mock_i_token_unlock_schedule_proxy.go -package=mocks

type ITokenUnlockScheduleProxy interface {
	SourceName() string
	// FindTokenUnlockEvents answers per coin symbol for the coins it covers; uncovered coins are absent.
	FindTokenUnlockEvents(executionContext context.Context, coinUnlockLookups []vo.CoinUnlockLookupVo) (map[string][]vo.TokenUnlockEventVo, error)
}
