package job

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
)

// HuntPipelineJob runs a hunt round at start and then every interval; a round still running when the next tick comes
// is left alone, so two rounds never overlap. Stopping lets the step in flight finish and starts no further one.
type HuntPipelineJob struct {
	huntPipelineApplication domaininterface.IHuntPipelineApplication
	interval                time.Duration
	stopRequested           chan struct{}
	stopOnce                sync.Once
	finished                chan struct{}
	roundRunning            atomic.Bool
	rounds                  sync.WaitGroup
}

func NewHuntPipelineJob(huntPipelineApplication domaininterface.IHuntPipelineApplication, interval time.Duration) *HuntPipelineJob {
	return &HuntPipelineJob{
		huntPipelineApplication: huntPipelineApplication,
		interval:                interval,
		stopRequested:           make(chan struct{}),
		finished:                make(chan struct{}),
	}
}

func (huntPipelineJob *HuntPipelineJob) Start(executionContext context.Context) {
	go func() {
		defer close(huntPipelineJob.finished)
		defer huntPipelineJob.rounds.Wait()

		ticker := time.NewTicker(huntPipelineJob.interval)
		defer ticker.Stop()
		huntPipelineJob.startRound(executionContext)
		for {
			select {
			case <-ticker.C:
				huntPipelineJob.startRound(executionContext)
			case <-huntPipelineJob.stopRequested:
				return
			case <-executionContext.Done():
				return
			}
		}
	}()
}

// startRound runs a round in the background unless one is still running; the flag is what keeps rounds from overlapping.
func (huntPipelineJob *HuntPipelineJob) startRound(executionContext context.Context) {
	if !huntPipelineJob.roundRunning.CompareAndSwap(false, true) {
		log.Println("hunt round skipped: the previous round is still running")
		return
	}
	huntPipelineJob.rounds.Add(1)
	go func() {
		defer huntPipelineJob.rounds.Done()
		defer huntPipelineJob.roundRunning.Store(false)

		// One line per round, enough to trace which run of which step stopped it.
		huntRound := huntPipelineJob.huntPipelineApplication.RunHuntRound(executionContext, huntPipelineJob.stopRequested, vo.PipelineRunTriggerSourceJob)
		pipelineRunIDs := make([]uint, 0, len(huntRound.Steps))
		for _, step := range huntRound.Steps {
			pipelineRunIDs = append(pipelineRunIDs, step.ID)
		}
		if huntRound.Completed {
			log.Printf("hunt round completed: pipeline runs %v", pipelineRunIDs)
			return
		}
		log.Printf("hunt round stopped at %s (%s): pipeline runs %v", huntRound.StoppedStep, huntRound.StoppedReason, pipelineRunIDs)
	}()
}

func (huntPipelineJob *HuntPipelineJob) Stop() {
	huntPipelineJob.stopOnce.Do(func() { close(huntPipelineJob.stopRequested) })
}

func (huntPipelineJob *HuntPipelineJob) Finished() <-chan struct{} {
	return huntPipelineJob.finished
}
