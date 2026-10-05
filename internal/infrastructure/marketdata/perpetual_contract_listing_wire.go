package marketdata

type binanceExchangeInformationWire struct {
	Symbols *[]struct {
		BaseAsset    string `json:"baseAsset"`
		QuoteAsset   string `json:"quoteAsset"`
		ContractType string `json:"contractType"`
		Status       string `json:"status"`
	} `json:"symbols"`
}

type bybitInstrumentsWire struct {
	ReturnCode    *int   `json:"retCode"`
	ReturnMessage string `json:"retMsg"`
	Result        struct {
		List *[]struct {
			BaseCoin     string `json:"baseCoin"`
			QuoteCoin    string `json:"quoteCoin"`
			ContractType string `json:"contractType"`
			Status       string `json:"status"`
			// SymbolType is empty or "innovation" for crypto; stock, ETF, commodity and forex contracts say so.
			SymbolType string `json:"symbolType"`
		} `json:"list"`
		NextPageCursor string `json:"nextPageCursor"`
	} `json:"result"`
}

type okxInstrumentsWire struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
	Data    *[]struct {
		InstrumentFamily string `json:"instFamily"`
		SettleCurrency   string `json:"settleCcy"`
		State            string `json:"state"`
		// InstrumentCategory "1" is crypto; other categories are stocks and other traditional assets.
		InstrumentCategory string `json:"instCategory"`
	} `json:"data"`
}
