package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_hunt_verdict_strategist_proxy.go -destination=mocks/mock_i_hunt_verdict_strategist_proxy.go -package=mocks

// IHuntVerdictStrategistProxy is the AI chief investment officer that weighs every coin of a round at once.
type IHuntVerdictStrategistProxy interface {
	// SynthesizeVerdicts returns domains.ErrHuntVerdictAnswerUnusable when the answer cannot be read; asking again may help.
	SynthesizeVerdicts(executionContext context.Context, materials []vo.HuntVerdictMaterialVo) ([]vo.HuntVerdictAnswerVo, error)
}
