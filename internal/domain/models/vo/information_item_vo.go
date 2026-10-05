package vo

import "time"

// InformationItemVo is one message an information source published, normalized out of the source's wire format.
type InformationItemVo struct {
	SourceName string
	// ExternalIdentifier is the source's own id or link for the message, so the same message is recognized across rounds.
	ExternalIdentifier string
	Title              string
	Link               string
	// PublishedAt is nil when the source does not say when (trending lists, token boards).
	PublishedAt *time.Time
	// DeclaredCoinSymbols is filled by structured sources; announcements leave it empty and the title is read instead.
	DeclaredCoinSymbols []string
	IsTraditionalAsset  bool
	// ChainID and ContractAddress are declared by on-chain sources, so later steps can identify the token exactly.
	ChainID         string
	ContractAddress string
}
