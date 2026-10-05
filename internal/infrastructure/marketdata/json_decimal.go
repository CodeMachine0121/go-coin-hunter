package marketdata

import (
	"bytes"
	"fmt"

	"github.com/shopspring/decimal"
)

// jsonDecimal reads a JSON number, or a number written as a string, without passing through float64.
type jsonDecimal struct {
	value decimal.Decimal
}

func (target *jsonDecimal) UnmarshalJSON(raw []byte) error {
	text := string(bytes.Trim(raw, `"`))
	parsed, parseError := decimal.NewFromString(text)
	if parseError != nil {
		return fmt.Errorf("read %s as a number: %w", raw, parseError)
	}
	target.value = parsed

	return nil
}

// decimalOrNil turns an absent or null figure into nil, so "unknown" never reads as zero.
func (target *jsonDecimal) decimalOrNil() *decimal.Decimal {
	if target == nil {
		return nil
	}
	value := target.value

	return &value
}

// changeRatioTo is latest ÷ this − 1; it is unknown unless both readings exist and this one is positive.
func (oldest *jsonDecimal) changeRatioTo(latest *jsonDecimal) *decimal.Decimal {
	if oldest == nil || latest == nil || !oldest.value.IsPositive() {
		return nil
	}
	ratio := latest.value.Div(oldest.value).Sub(decimal.NewFromInt(1))

	return &ratio
}
