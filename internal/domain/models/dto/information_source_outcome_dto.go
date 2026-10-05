package dto

type InformationSourceOutcomeDto struct {
	SourceName           string `json:"sourceName"`
	Succeeded            bool   `json:"succeeded"`
	InformationItemCount int    `json:"informationItemCount"`
	FailureReason        string `json:"failureReason"`
}
