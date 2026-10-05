package main

import (
	"net/http"
	"testing"
	"time"

	domaininterface "github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/job"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestShutDownWaitsForJobsWithinTheGracePeriod(t *testing.T) {
	backgroundJob := mocks.NewMockIBackgroundJob(gomock.NewController(t))
	finished := make(chan struct{})
	backgroundJob.EXPECT().Stop().Do(func() { close(finished) })
	backgroundJob.EXPECT().Finished().Return(finished)
	abandoned := false

	hadToAbandon := shutDown(&http.Server{}, job.NewBackgroundJobManager([]domaininterface.IBackgroundJob{backgroundJob}), time.Second, func() { abandoned = true })

	assert.False(t, hadToAbandon)
	assert.False(t, abandoned)
}

func TestShutDownAbandonsJobsWhenTheGracePeriodRunsOut(t *testing.T) {
	backgroundJob := mocks.NewMockIBackgroundJob(gomock.NewController(t))
	backgroundJob.EXPECT().Stop()
	backgroundJob.EXPECT().Finished().Return(make(chan struct{}))
	abandoned := false
	startedAt := time.Now()

	hadToAbandon := shutDown(&http.Server{}, job.NewBackgroundJobManager([]domaininterface.IBackgroundJob{backgroundJob}), 50*time.Millisecond, func() { abandoned = true })

	assert.True(t, hadToAbandon)
	assert.True(t, abandoned)
	assert.GreaterOrEqual(t, time.Since(startedAt), 50*time.Millisecond)
}
