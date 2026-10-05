package job_test

import (
	"context"
	"testing"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/job"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBackgroundJobManagerStartsStopsAndWaitsForEveryJob(t *testing.T) {
	controller := gomock.NewController(t)
	finished := make(chan struct{})
	close(finished)
	backgroundJobs := []domaininterface.IBackgroundJob{}
	for range 2 {
		backgroundJob := mocks.NewMockIBackgroundJob(controller)
		backgroundJob.EXPECT().Start(gomock.Any())
		backgroundJob.EXPECT().Stop()
		backgroundJob.EXPECT().Finished().Return(finished)
		backgroundJobs = append(backgroundJobs, backgroundJob)
	}
	backgroundJobManager := job.NewBackgroundJobManager(backgroundJobs)

	backgroundJobManager.StartAll(context.Background())
	backgroundJobManager.StopAll()

	assert.True(t, backgroundJobManager.WaitAll(context.Background()))
}

func TestBackgroundJobManagerGivesUpWaitingWhenTheContextEnds(t *testing.T) {
	controller := gomock.NewController(t)
	backgroundJob := mocks.NewMockIBackgroundJob(controller)
	backgroundJob.EXPECT().Finished().Return(make(chan struct{}))
	expired, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, job.NewBackgroundJobManager([]domaininterface.IBackgroundJob{backgroundJob}).WaitAll(expired))
}
