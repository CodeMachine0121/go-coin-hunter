package marketdata

type goPlusResponseWire[T any] struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Result  map[string]T `json:"result"`
}

// goPlusEvmTokenSecurityWire flags are "1" or "0"; taxes are ratios as text, empty when unknown.
type goPlusEvmTokenSecurityWire struct {
	IsHoneypot           string `json:"is_honeypot"`
	CannotSellAll        string `json:"cannot_sell_all"`
	BuyTax               string `json:"buy_tax"`
	SellTax              string `json:"sell_tax"`
	IsMintable           string `json:"is_mintable"`
	TransferPausable     string `json:"transfer_pausable"`
	IsBlacklisted        string `json:"is_blacklisted"`
	OwnerAddress         string `json:"owner_address"`
	HiddenOwner          string `json:"hidden_owner"`
	CanTakeBackOwnership string `json:"can_take_back_ownership"`
}

type goPlusSolanaAuthorityWire struct {
	Status string `json:"status"`
}

type goPlusSolanaTokenSecurityWire struct {
	Mintable        goPlusSolanaAuthorityWire `json:"mintable"`
	Freezable       goPlusSolanaAuthorityWire `json:"freezable"`
	NonTransferable string                    `json:"non_transferable"`
}
