package marketdata

type defiLlamaEmissionWire struct {
	Name     string `json:"name"`
	GeckoID  string `json:"gecko_id"`
	Metadata *struct {
		Events []struct {
			Timestamp  int64          `json:"timestamp"`
			NoOfTokens []*jsonDecimal `json:"noOfTokens"`
		} `json:"events"`
	} `json:"metadata"`
}
