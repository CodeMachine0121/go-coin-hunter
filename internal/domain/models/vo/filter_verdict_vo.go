package vo

type FilterOutcomeVo string

const (
	FilterOutcomePassed   FilterOutcomeVo = "passed"
	FilterOutcomeRejected FilterOutcomeVo = "rejected"
	FilterOutcomeNoData   FilterOutcomeVo = "noData"
)

// FilterVerdictVo is one rule's judgement on one candidate; rejected and noData always carry a reason.
type FilterVerdictVo struct {
	FilterName string
	Outcome    FilterOutcomeVo
	Reason     string
}
