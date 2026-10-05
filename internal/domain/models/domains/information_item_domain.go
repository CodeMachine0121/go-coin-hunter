package domains

import (
	"strings"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// InformationItemDomain turns one source message into the intelligence the system keeps: one per coin it names, or one with no coin.
type InformationItemDomain struct {
	informationItem vo.InformationItemVo
}

func NewInformationItemDomain(informationItem vo.InformationItemVo) InformationItemDomain {
	return InformationItemDomain{informationItem: informationItem}
}

// CoinIntelligences reads the declared coins first and the title only when none were declared; a message with no publish time counts as published when first received.
func (informationItemDomain InformationItemDomain) CoinIntelligences(
	pipelineRunID uint, receivedAt time.Time,
) []entities.CoinIntelligence {
	informationItem := informationItemDomain.informationItem
	publishedAt := receivedAt
	if informationItem.PublishedAt != nil {
		publishedAt = informationItem.PublishedAt.UTC()
	}

	coinSymbols := []string{}
	for _, declaredCoinSymbol := range informationItem.DeclaredCoinSymbols {
		coinSymbol := strings.ToUpper(strings.TrimSpace(declaredCoinSymbol))
		if coinSymbol != "" {
			coinSymbols = append(coinSymbols, coinSymbol)
		}
	}
	if len(coinSymbols) == 0 {
		coinSymbols = NewAnnouncementTitleDomain(informationItem.Title).CoinSymbols()
	}
	if len(coinSymbols) == 0 {
		coinSymbols = []string{""}
	}

	coinIntelligences := make([]entities.CoinIntelligence, 0, len(coinSymbols))
	for _, coinSymbol := range coinSymbols {
		coinIntelligences = append(coinIntelligences, entities.CoinIntelligence{
			PipelineRunID:      pipelineRunID,
			SourceName:         informationItem.SourceName,
			ExternalIdentifier: informationItem.ExternalIdentifier,
			CoinSymbol:         coinSymbol,
			Title:              informationItem.Title,
			Link:               informationItem.Link,
			PublishedAt:        publishedAt,
			IsTraditionalAsset: informationItem.IsTraditionalAsset,
		})
	}

	return coinIntelligences
}
