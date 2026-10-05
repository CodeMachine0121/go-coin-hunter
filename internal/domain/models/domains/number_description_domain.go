package domains

import (
	"strings"

	"github.com/shopspring/decimal"
)

var (
	tenThousand    = decimal.NewFromInt(10_000)
	hundredMillion = decimal.NewFromInt(100_000_000)
	hundred        = decimal.NewFromInt(100)
)

// NumberDescriptionDomain writes a figure the way a Taiwanese trader reads it in a reason: 52 億、1,000 萬、19%.
type NumberDescriptionDomain struct {
	value decimal.Decimal
}

func NewNumberDescriptionDomain(value decimal.Decimal) NumberDescriptionDomain {
	return NumberDescriptionDomain{value: value}
}

// AsUsd reads "52 億美元", "1,000 萬美元" or "999 美元".
func (numberDescriptionDomain NumberDescriptionDomain) AsUsd() string {
	figure, unit := numberDescriptionDomain.scaled()
	if unit == "" {
		return figure + " 美元"
	}

	return figure + " " + unit + "美元"
}

// AsTokenQuantity reads "10 億", "5,000 萬" or "999".
func (numberDescriptionDomain NumberDescriptionDomain) AsTokenQuantity() string {
	figure, unit := numberDescriptionDomain.scaled()
	if unit == "" {
		return figure
	}

	return figure + " " + unit
}

// AsPercentage reads the value as a ratio: 0.105 is 10.5%.
func (numberDescriptionDomain NumberDescriptionDomain) AsPercentage() string {
	return numberDescriptionDomain.value.Mul(hundred).Round(2).String() + "%"
}

// scaled picks 億 from a hundred million and 萬 from ten thousand, keeps at most two decimals and groups thousands.
func (numberDescriptionDomain NumberDescriptionDomain) scaled() (string, string) {
	scaledValue, unit := numberDescriptionDomain.value, ""
	switch absolute := numberDescriptionDomain.value.Abs(); {
	case absolute.GreaterThanOrEqual(hundredMillion):
		scaledValue, unit = numberDescriptionDomain.value.Div(hundredMillion), "億"
	case absolute.GreaterThanOrEqual(tenThousand):
		scaledValue, unit = numberDescriptionDomain.value.Div(tenThousand), "萬"
	}

	text := scaledValue.Round(2).String()
	sign := ""
	if strings.HasPrefix(text, "-") {
		sign, text = "-", strings.TrimPrefix(text, "-")
	}
	integerPart, fractionPart, hasFraction := strings.Cut(text, ".")
	grouped := strings.Builder{}
	for index, digit := range integerPart {
		if index > 0 && (len(integerPart)-index)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(digit)
	}
	if hasFraction {
		return sign + grouped.String() + "." + fractionPart, unit
	}

	return sign + grouped.String(), unit
}
