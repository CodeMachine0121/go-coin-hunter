package _interface

import "github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"

//go:generate go tool mockgen -source=i_coin_candidate_filter_handler.go -destination=mocks/mock_i_coin_candidate_filter_handler.go -package=mocks

// ICoinCandidateFilterHandler is one filtering rule; every rule has its own handler and the composition root injects
// them as a list, so a rule is added or removed without touching the others.
type ICoinCandidateFilterHandler interface {
	FilterName() string
	// Evaluate judges one candidate from its profile alone; it never reaches outside the profile.
	Evaluate(coinProfile vo.CoinProfileVo) vo.FilterVerdictVo
}
