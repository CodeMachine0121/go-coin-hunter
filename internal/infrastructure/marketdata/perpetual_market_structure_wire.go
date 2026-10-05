package marketdata

type binanceTicker24hWire struct {
	LastPrice          *jsonDecimal `json:"lastPrice"`
	PriceChangePercent *jsonDecimal `json:"priceChangePercent"`
	QuoteVolume        *jsonDecimal `json:"quoteVolume"`
}

type binancePremiumIndexWire struct {
	LastFundingRate *jsonDecimal `json:"lastFundingRate"`
}

type binanceOpenInterestHistoryWire struct {
	SumOpenInterestValue *jsonDecimal `json:"sumOpenInterestValue"`
	Timestamp            int64        `json:"timestamp"`
}

type bybitTickersWire struct {
	ReturnCode *int `json:"retCode"`
	Result     struct {
		List []struct {
			LastPrice         *jsonDecimal `json:"lastPrice"`
			Price24hPercent   *jsonDecimal `json:"price24hPcnt"`
			Turnover24h       *jsonDecimal `json:"turnover24h"`
			FundingRate       *jsonDecimal `json:"fundingRate"`
			OpenInterestValue *jsonDecimal `json:"openInterestValue"`
		} `json:"list"`
	} `json:"result"`
}

type bybitOpenInterestWire struct {
	ReturnCode *int `json:"retCode"`
	Result     struct {
		List []bybitOpenInterestReadingWire `json:"list"`
	} `json:"result"`
}

// bybitOpenInterestReadingWire timestamps are milliseconds written as text.
type bybitOpenInterestReadingWire struct {
	OpenInterest *jsonDecimal `json:"openInterest"`
	Timestamp    jsonDecimal  `json:"timestamp"`
}

type okxResponseWire[T any] struct {
	Code string `json:"code"`
	Data []T    `json:"data"`
}

type okxTickerWire struct {
	Last              *jsonDecimal `json:"last"`
	Open24h           *jsonDecimal `json:"open24h"`
	VolumeCurrency24h *jsonDecimal `json:"volCcy24h"`
}

type okxFundingRateWire struct {
	FundingRate *jsonDecimal `json:"fundingRate"`
}

type okxOpenInterestWire struct {
	OpenInterestUsd *jsonDecimal `json:"oiUsd"`
}
