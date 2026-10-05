package job_test

import (
	"bytes"
	"context"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
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
	roundTimes := []time.Time{}
	roundTimesLock := sync.Mutex{}
	huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), vo.PipelineRunTriggerSourceJob).DoAndReturn(
		func(context.Context, <-chan struct{}, vo.PipelineRunTriggerSourceVo) (dto.HuntRoundDto, error) {
			roundTimesLock.Lock()
			defer roundTimesLock.Unlock()
			roundTimes = append(roundTimes, time.Now())
			return dto.HuntRoundDto{Completed: true}, nil
		}).MinTimes(3)
	interval := 60 * time.Millisecond
	huntPipelineJob := job.NewHuntPipelineJob(huntPipeline, interval)
	startedAt := time.Now()

	huntPipelineJob.Start(context.Background())
	roundCount := func() int {
		roundTimesLock.Lock()
		defer roundTimesLock.Unlock()
		return len(roundTimes)
	}
	waitUntil(t, func() bool { return roundCount() >= 3 })
	huntPipelineJob.Stop()
	<-huntPipelineJob.Finished()

	roundTimesLock.Lock()
	defer roundTimesLock.Unlock()
	assert.Less(t, roundTimes[0].Sub(startedAt), interval/2, "the first round starts without waiting for the interval")
	for index := 1; index < len(roundTimes); index++ {
		assert.GreaterOrEqual(t, roundTimes[index].Sub(roundTimes[index-1]), interval*3/4, "rounds are an interval apart")
	}
}

func TestHuntPipelineJobKeepsItsScheduleWhenARoundIsRefused(t *testing.T) {
	controller := gomock.NewController(t)
	huntPipeline := mocks.NewMockIHuntPipelineApplication(controller)
	rounds := atomic.Int32{}
	huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, <-chan struct{}, vo.PipelineRunTriggerSourceVo) (dto.HuntRoundDto, error) {
			rounds.Add(1)
			return dto.HuntRoundDto{}, domains.ErrHuntRoundAlreadyRunning
		}).MinTimes(2)
	huntPipelineJob := job.NewHuntPipelineJob(huntPipeline, 10*time.Millisecond)
	logBuffer := &bytes.Buffer{}
	previousWriter := log.Writer()
	log.SetOutput(logBuffer)
	t.Cleanup(func() { log.SetOutput(previousWriter) })

	huntPipelineJob.Start(context.Background())
	waitUntil(t, func() bool { return rounds.Load() >= 2 })
	huntPipelineJob.Stop()

	<-huntPipelineJob.Finished()
	assert.Contains(t, logBuffer.String(), "hunt round skipped: 已有獵捕回合進行中")
}

func TestHuntPipelineJobStopLetsTheRoundFinishItsStep(t *testing.T) {
	controller := gomock.NewController(t)
	huntPipeline := mocks.NewMockIHuntPipelineApplication(controller)
	inRound := make(chan struct{})
	sawStop := atomic.Bool{}
	huntPipeline.EXPECT().RunHuntRound(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(roundContext context.Context, stopBetweenSteps <-chan struct{}, _ vo.PipelineRunTriggerSourceVo) (dto.HuntRoundDto, error) {
			close(inRound)
			<-stopBetweenSteps
			sawStop.Store(true)
			assert.NoError(t, roundContext.Err(), "stopping must not abandon the step in flight")
			return dto.HuntRoundDto{StoppedStep: "insight", StoppedReason: "洞察未開始：服務關閉中"}, nil
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
		func(roundContext context.Context, _ <-chan struct{}, _ vo.PipelineRunTriggerSourceVo) (dto.HuntRoundDto, error) {
			<-roundContext.Done()
			return dto.HuntRoundDto{StoppedStep: "discovery", StoppedReason: "context canceled"}, nil
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
