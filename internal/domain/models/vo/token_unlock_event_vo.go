package vo

import (
	"time"

	"github.com/shopspring/decimal"
)

type TokenUnlockEventVo struct {
	UnlockAt time.Time
	Amount   decimal.Decimal
}
