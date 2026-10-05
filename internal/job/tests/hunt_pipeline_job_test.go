package job_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func waitUntil(t *testing.T, condition func() bool) {
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		require.True(t, time.Now().Before(deadline), "condition never held")
		time.Sleep(5 * time.Millisecond)
	}
}

func TestHuntPipelineJobRunsAtStartThenEveryInterval(t *testing.T) {
	controller := gomock.NewController(t)
	huntPipeline := mocks.NewMockIHuntPipelineApplication(controller)
	rounds := atomic.Int32{}
	huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), vo.PipelineRunTriggerSourceJob).DoAndReturn(
		func(context.Context, <-chan struct{}, vo.PipelineRunTriggerSourceVo) dto.HuntRoundDto {
			rounds.Add(1)
			return dto.HuntRoundDto{Completed: true, Steps: []dto.PipelineRunDto{{ID: 1}}}
		}).MinTimes(3)
	huntPipelineJob := job.NewHuntPipelineJob(huntPipeline, 30*time.Millisecond)
	startedAt := time.Now()

	huntPipelineJob.Start(context.Background())
	waitUntil(t, func() bool { return rounds.Load() >= 1 })
	assert.Less(t, time.Since(startedAt), 30*time.Millisecond, "the first round starts without waiting for the interval")
	waitUntil(t, func() bool { return rounds.Load() >= 3 })
	huntPipelineJob.Stop()

	select {
	case <-huntPipelineJob.Finished():
	case <-time.After(time.Second):
		t.Fatal("job did not finish after stop")
	}
}

func TestHuntPipelineJobNeverOverlapsRounds(t *testing.T) {
	controller := gomock.NewController(t)
	huntPipeline := mocks.NewMockIHuntPipelineApplication(controller)
	release := make(chan struct{})
	running, mostAtOnce, rounds := atomic.Int32{}, atomic.Int32{}, atomic.Int32{}
	huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, <-chan struct{}, vo.PipelineRunTriggerSourceVo) dto.HuntRoundDto {
			rounds.Add(1)
			if current := running.Add(1); current > mostAtOnce.Load() {
				mostAtOnce.Store(current)
			}
			<-release
			running.Add(-1)
			return dto.HuntRoundDto{StoppedStep: "discovery", StoppedReason: "探索未成功：noData"}
		}).AnyTimes()
	huntPipelineJob := job.NewHuntPipelineJob(huntPipeline, 10*time.Millisecond)

	huntPipelineJob.Start(context.Background())
	time.Sleep(80 * time.Millisecond)

	assert.Equal(t, int32(1), rounds.Load(), "ticks during a running round are skipped")
	close(release)
	waitUntil(t, func() bool { return rounds.Load() >= 2 })
	huntPipelineJob.Stop()
	<-huntPipelineJob.Finished()
	assert.Equal(t, int32(1), mostAtOnce.Load())
}

func TestHuntPipelineJobStopLetsTheRoundFinishItsStep(t *testing.T) {
	controller := gomock.NewController(t)
	huntPipeline := mocks.NewMockIHuntPipelineApplication(controller)
	inRound := make(chan struct{})
	sawStop := atomic.Bool{}
	huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(roundContext context.Context, stopBetweenSteps <-chan struct{}, _ vo.PipelineRunTriggerSourceVo) dto.HuntRoundDto {
			close(inRound)
			<-stopBetweenSteps
			sawStop.Store(true)
			assert.NoError(t, roundContext.Err(), "stopping must not abandon the step in flight")
			return dto.HuntRoundDto{StoppedStep: "insight", StoppedReason: "洞察未開始：服務關閉中"}
		})
	huntPipelineJob := job.NewHuntPipelineJob(huntPipeline, time.Hour)

	huntPipelineJob.Start(context.Background())
	<-inRound
	huntPipelineJob.Stop()
	huntPipelineJob.Stop()

	select {
	case <-huntPipelineJob.Finished():
	case <-time.After(time.Second):
		t.Fatal("job did not finish after stop")
	}
	assert.True(t, sawStop.Load())
}

func TestHuntPipelineJobEndsWithItsContext(t *testing.T) {
	controller := gomock.NewController(t)
	huntPipeline := mocks.NewMockIHuntPipelineApplication(controller)
	huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(roundContext context.Context, _ <-chan struct{}, _ vo.PipelineRunTriggerSourceVo) dto.HuntRoundDto {
			<-roundContext.Done()
			return dto.HuntRoundDto{StoppedStep: "discovery", StoppedReason: "context canceled"}
		})
	executionContext, cancel := context.WithCancel(context.Background())
	huntPipelineJob := job.NewHuntPipelineJob(huntPipeline, time.Hour)

	huntPipelineJob.Start(executionContext)
	cancel()

	select {
	case <-huntPipelineJob.Finished():
	case <-time.After(time.Second):
		t.Fatal("job did not finish when its context ended")
	}
}
