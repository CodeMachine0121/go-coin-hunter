package marketdata

type dexScreenerPairWire struct {
	BaseToken struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	} `json:"baseToken"`
	FullyDilutedValuation *jsonDecimal `json:"fdv"`
	Liquidity             *struct {
		Usd *jsonDecimal `json:"usd"`
	} `json:"liquidity"`
	Volume *struct {
		Hours24 *jsonDecimal `json:"h24"`
	} `json:"volume"`
}
