package informationsource

type dexScreenerTokenProfileWire struct {
	Url          string `json:"url"`
	ChainID      string `json:"chainId"`
	TokenAddress string `json:"tokenAddress"`
}

type dexScreenerPairWire struct {
	ChainID   string `json:"chainId"`
	BaseToken struct {
		Address string `json:"address"`
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
	} `json:"baseToken"`
	PairCreatedAt int64 `json:"pairCreatedAt"`
}
