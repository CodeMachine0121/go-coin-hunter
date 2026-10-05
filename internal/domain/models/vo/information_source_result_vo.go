package vo

// InformationSourceResultVo is what one information source returned in one round: its items, or why it failed.
type InformationSourceResultVo struct {
	SourceName       string
	InformationItems []InformationItemVo
	// FailureReason is empty when the source answered.
	FailureReason string
}
