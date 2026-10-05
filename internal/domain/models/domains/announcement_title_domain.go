package domains

import (
	"regexp"
	"slices"
	"strings"
)

// Tickers are 2-15 upper-case letters or digits; "(CT)" names a coin, "CTUSDT" names its USDT market.
var (
	parenthesizedCoinSymbolPattern = regexp.MustCompile(`\(([A-Z0-9]{2,15})\)`)
	usdtMarketCoinSymbolPattern    = regexp.MustCompile(`\b([A-Z0-9]{2,15})USDT\b`)
)

// AnnouncementTitleDomain reads which coins an exchange announcement title names.
type AnnouncementTitleDomain struct {
	title string
}

func NewAnnouncementTitleDomain(title string) AnnouncementTitleDomain {
	return AnnouncementTitleDomain{title: title}
}

// CoinSymbols lists every coin the title names, once each, in the order they first appear.
func (announcementTitleDomain AnnouncementTitleDomain) CoinSymbols() []string {
	type positionedCoinSymbol struct {
		position   int
		coinSymbol string
	}

	positionedCoinSymbols := []positionedCoinSymbol{}
	for _, pattern := range []*regexp.Regexp{parenthesizedCoinSymbolPattern, usdtMarketCoinSymbolPattern} {
		for _, match := range pattern.FindAllStringSubmatchIndex(announcementTitleDomain.title, -1) {
			positionedCoinSymbols = append(positionedCoinSymbols, positionedCoinSymbol{
				position:   match[2],
				coinSymbol: announcementTitleDomain.title[match[2]:match[3]],
			})
		}
	}

	slices.SortStableFunc(positionedCoinSymbols, func(left, right positionedCoinSymbol) int {
		return left.position - right.position
	})

	seenCoinSymbols := map[string]bool{}
	coinSymbols := []string{}
	for _, positioned := range positionedCoinSymbols {
		coinSymbol := strings.ToUpper(positioned.coinSymbol)
		// "(2026)" is a year in a title, not a ticker.
		if seenCoinSymbols[coinSymbol] || strings.Trim(coinSymbol, "0123456789") == "" {
			continue
		}
		seenCoinSymbols[coinSymbol] = true
		coinSymbols = append(coinSymbols, coinSymbol)
	}

	return coinSymbols
}
