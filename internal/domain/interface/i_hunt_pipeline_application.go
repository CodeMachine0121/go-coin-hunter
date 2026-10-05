package _interface

import (
	"context"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

//go:generate go tool mockgen -source=i_hunt_pipeline_application.go -destination=mocks/mock_i_hunt_pipeline_application.go -package=mocks

// IHuntPipelineApplication runs a whole hunt round; the scheduled job reaches the pipeline only through it.
type IHuntPipelineApplication interface {
	// RunHuntRound starts no further step once stopBetweenSteps is closed, letting the step in flight finish; a nil
	// channel never stops the round. An ended context abandons the step in flight as well.
	// RunHuntRound refuses with domains.ErrHuntRoundAlreadyRunning while another round, scheduled or manual, is running.
	RunHuntRound(executionContext context.Context, stopBetweenSteps <-chan struct{}, triggerSource vo.PipelineRunTriggerSourceVo) (dto.HuntRoundDto, error)
}
