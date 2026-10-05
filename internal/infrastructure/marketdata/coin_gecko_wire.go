package marketdata

type coinGeckoListedCoinWire struct {
	ID        string            `json:"id"`
	Symbol    string            `json:"symbol"`
	Name      string            `json:"name"`
	Platforms map[string]string `json:"platforms"`
}

// coinGeckoMarketWire figures are JSON numbers that may be null; they are kept as raw text so no precision is lost.
type coinGeckoMarketWire struct {
	ID                    string       `json:"id"`
	Symbol                string       `json:"symbol"`
	Name                  string       `json:"name"`
	MarketCap             *jsonDecimal `json:"market_cap"`
	FullyDilutedValuation *jsonDecimal `json:"fully_diluted_valuation"`
	TotalVolume           *jsonDecimal `json:"total_volume"`
	CirculatingSupply     *jsonDecimal `json:"circulating_supply"`
	TotalSupply           *jsonDecimal `json:"total_supply"`
	MaxSupply             *jsonDecimal `json:"max_supply"`
}
