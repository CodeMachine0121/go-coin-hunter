package informationsource

type coinGeckoTrendingWire struct {
	Coins []struct {
		Item struct {
			ID     string `json:"id"`
			Symbol string `json:"symbol"`
			Name   string `json:"name"`
		} `json:"item"`
	} `json:"coins"`
}
