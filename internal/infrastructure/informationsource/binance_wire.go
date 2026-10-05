package informationsource

type binanceAnnouncementListWire struct {
	Code string `json:"code"`
	Data struct {
		Catalogs []struct {
			Articles []binanceArticleWire `json:"articles"`
		} `json:"catalogs"`
	} `json:"data"`
}

type binanceArticleWire struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	ReleaseDate int64  `json:"releaseDate"`
}

type binanceExchangeInformationWire struct {
	Symbols []binanceContractSymbolWire `json:"symbols"`
}

type binanceContractSymbolWire struct {
	Symbol       string `json:"symbol"`
	BaseAsset    string `json:"baseAsset"`
	ContractType string `json:"contractType"`
	Status       string `json:"status"`
	OnboardDate  int64  `json:"onboardDate"`
}
