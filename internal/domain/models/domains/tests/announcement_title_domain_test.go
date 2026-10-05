package domains_test

import (
	"testing"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/stretchr/testify/assert"
)

func TestAnnouncementTitleCoinSymbols(t *testing.T) {
	testCases := []struct {
		name  string
		title string
		want  []string
	}{
		{name: "a parenthesized ticker", title: "Binance Will List Cotton (CT)", want: []string{"CT"}},
		{name: "a USDT market name", title: "New listing: CTUSDT Perpetual Contract", want: []string{"CT"}},
		{name: "several coins in title order, once each", title: "Binance Will List Zora (ZORA) and Pump (PUMP); ZORAUSDT opens", want: []string{"ZORA", "PUMP"}},
		{name: "a date in parentheses is not a ticker", title: "Binance Futures Will Launch USDⓈ-Margined CTUSDT Perpetual Contract (2026-10-01)", want: []string{"CT"}},
		{name: "a year alone in parentheses is not a ticker", title: "Listing roundup (2026)", want: []string{}},
		{name: "no ticker at all", title: "系統維護通知", want: []string{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, domains.NewAnnouncementTitleDomain(testCase.title).CoinSymbols())
		})
	}
}
