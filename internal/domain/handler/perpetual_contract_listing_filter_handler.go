package handler

import (
	"strings"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// PerpetualContractListingFilterHandler keeps only coins the trader can actually trade: those with a USDT perpetual somewhere.
type PerpetualContractListingFilterHandler struct{}

func NewPerpetualContractListingFilterHandler() *PerpetualContractListingFilterHandler {
	return &PerpetualContractListingFilterHandler{}
}

func (perpetualContractListingFilterHandler *PerpetualContractListingFilterHandler) FilterName() string {
	return "perpetualContractListing"
}

func (perpetualContractListingFilterHandler *PerpetualContractListingFilterHandler) Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo {
	if len(coinProfile.PerpetualContractExchanges) == 0 {
		return vo.FilterVerdictVo{FilterName: perpetualContractListingFilterHandler.FilterName(), Outcome: vo.FilterOutcomeRejected,
			Reason: "幣安、Bybit、OKX 皆無 USDT 永續合約"}
	}

	return vo.FilterVerdictVo{FilterName: perpetualContractListingFilterHandler.FilterName(), Outcome: vo.FilterOutcomePassed,
		Reason: "已上 " + strings.Join(coinProfile.PerpetualContractExchanges, "、") + " USDT 永續合約"}
}
