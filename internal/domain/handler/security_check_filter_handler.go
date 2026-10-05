package handler

import (
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/shopspring/decimal"
)

// SecurityCheckFilterHandler rejects contracts that trap holders: honeypots, unsellable tokens, punitive taxes,
// and supply or balances someone can still mint or freeze.
type SecurityCheckFilterHandler struct {
	maximumTaxRate decimal.Decimal
}

func NewSecurityCheckFilterHandler(maximumTaxRate decimal.Decimal) *SecurityCheckFilterHandler {
	return &SecurityCheckFilterHandler{maximumTaxRate: maximumTaxRate}
}

func (securityCheckFilterHandler *SecurityCheckFilterHandler) FilterName() string {
	return "securityCheck"
}

func (securityCheckFilterHandler *SecurityCheckFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	verdict := vo.FilterVerdictVo{FilterName: securityCheckFilterHandler.FilterName(), Outcome: vo.FilterOutcomeNoData}
	if coinProfile.ContractAddress == nil {
		verdict.Reason = "沒有主流鏈上的合約位址"
		return verdict
	}
	if coinProfile.TokenSecurity == nil {
		verdict.Reason = "查不到合約安全資料"
		if coinProfile.TokenSecurityNotQueriedReason != "" {
			verdict.Reason = coinProfile.TokenSecurityNotQueriedReason
		}
		return verdict
	}

	tokenSecurity := coinProfile.TokenSecurity
	maximumTaxRateText := domains.NewNumberDescriptionDomain(securityCheckFilterHandler.maximumTaxRate).AsPercentage()
	findings := []string{}
	if tokenSecurity.IsHoneypot {
		findings = append(findings, "蜜罐，無法賣出")
	}
	if tokenSecurity.CannotSell && !tokenSecurity.IsHoneypot {
		findings = append(findings, "無法賣出全部持幣")
	}
	if tokenSecurity.BuyTaxRate != nil && tokenSecurity.BuyTaxRate.GreaterThan(securityCheckFilterHandler.maximumTaxRate) {
		findings = append(findings, "買入稅 "+domains.NewNumberDescriptionDomain(*tokenSecurity.BuyTaxRate).AsPercentage()+" 超過上限 "+maximumTaxRateText)
	}
	if tokenSecurity.SellTaxRate != nil && tokenSecurity.SellTaxRate.GreaterThan(securityCheckFilterHandler.maximumTaxRate) {
		findings = append(findings, "賣出稅 "+domains.NewNumberDescriptionDomain(*tokenSecurity.SellTaxRate).AsPercentage()+" 超過上限 "+maximumTaxRateText)
	}
	if tokenSecurity.IsMintable {
		findings = append(findings, "發行方仍可增發")
	}
	if tokenSecurity.CanFreezeHolders {
		findings = append(findings, "發行方可凍結持有人資產")
	}

	if len(findings) > 0 {
		verdict.Outcome, verdict.Reason = vo.FilterOutcomeRejected, strings.Join(findings, "；")
		return verdict
	}
	verdict.Outcome, verdict.Reason = vo.FilterOutcomePassed, ""

	return verdict
}
