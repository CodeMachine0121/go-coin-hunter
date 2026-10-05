package dto

import "time"

type CoinIntelligenceDto struct {
	SourceName         string    `json:"sourceName"`
	CoinSymbol         string    `json:"coinSymbol"`
	Title              string    `json:"title"`
	Link               string    `json:"link"`
	PublishedAt        time.Time `json:"publishedAt"`
	IsTraditionalAsset bool      `json:"isTraditionalAsset"`
}
