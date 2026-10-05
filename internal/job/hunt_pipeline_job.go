package job

import (
	"context"
	"log"
	"sync"
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

// startRound runs a round in the background; the pipeline itself refuses a round while another is running, so a tick
// that comes during a long round is skipped rather than queued.
func (huntPipelineJob *HuntPipelineJob) startRound(executionContext context.Context) {
	huntPipelineJob.rounds.Add(1)
	go func() {
		defer huntPipelineJob.rounds.Done()

		if _, roundError := huntPipelineJob.huntPipelineApplication.RunHuntRound(
			executionContext, huntPipelineJob.stopRequested, vo.PipelineRunTriggerSourceJob); roundError != nil {
			log.Printf("hunt round skipped: %v", roundError)
		}
	}()
}

func (huntPipelineJob *HuntPipelineJob) Stop() {
	huntPipelineJob.stopOnce.Do(func() { close(huntPipelineJob.stopRequested) })
}

func (huntPipelineJob *HuntPipelineJob) Finished() <-chan struct{} {
	return huntPipelineJob.finished
}
