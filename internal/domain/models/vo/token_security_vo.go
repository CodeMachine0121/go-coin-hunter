package vo

import "github.com/shopspring/decimal"

// TokenSecurityVo is a contract's risk findings; a nil tax rate means the source could not tell.
type TokenSecurityVo struct {
	IsHoneypot  bool
	CannotSell  bool
	BuyTaxRate  *decimal.Decimal
	SellTaxRate *decimal.Decimal
	// IsMintable and CanFreezeHolders are only true while someone still controls the contract.
	IsMintable       bool
	CanFreezeHolders bool
}
