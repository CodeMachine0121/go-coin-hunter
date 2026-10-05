package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/application"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/interface/mocks"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/entities"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/vo"
	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var restartedAt = time.Date(2026, 10, 5, 13, 0, 0, 0, time.UTC)

func newPipelineRunApplication(t *testing.T) (*application.PipelineRunApplication, *mocks.MockIPipelineRunRepository) {
	controller := gomock.NewController(t)
	pipelineRunRepository := mocks.NewMockIPipelineRunRepository(controller)
	clockProxy := mocks.NewMockIClockProxy(controller)
	clockProxy.EXPECT().Now().Return(restartedAt).AnyTimes()

	return application.NewPipelineRunApplication(service.NewPipelineRunService(pipelineRunRepository, clockProxy)), pipelineRunRepository
}

func TestFailInterruptedPipelineRunsMarksRunningRunsFailed(t *testing.T) {
	pipelineRunApplication, pipelineRunRepository := newPipelineRunApplication(t)
	pipelineRunRepository.EXPECT().FindRunning(gomock.Any()).Return([]entities.PipelineRun{
		{ID: 3, Status: string(vo.PipelineRunStatusRunning)},
	}, nil)
	pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, pipelineRun entities.PipelineRun) error {
			assert.Equal(t, uint(3), pipelineRun.ID)
			assert.Equal(t, string(vo.PipelineRunStatusFailed), pipelineRun.Status)
			assert.Equal(t, domains.InterruptedByRestartReason, pipelineRun.FailureReason)
			assert.Equal(t, restartedAt, *pipelineRun.FinishedAt)
			return nil
		})

	failedCount, failError := pipelineRunApplication.FailInterruptedPipelineRuns(context.Background())

	require.NoError(t, failError)
	assert.Equal(t, 1, failedCount)
}

func TestGetPipelineRunsKeepsTheNewestFirstOrder(t *testing.T) {
	pipelineRunApplication, pipelineRunRepository := newPipelineRunApplication(t)
	pipelineRunRepository.EXPECT().FindAll(gomock.Any()).Return([]entities.PipelineRun{
		{ID: 2, StartedAt: restartedAt}, {ID: 1, StartedAt: restartedAt.Add(-time.Hour)},
	}, nil)

	pipelineRuns, findError := pipelineRunApplication.GetPipelineRuns(context.Background())

	require.NoError(t, findError)
	require.Len(t, pipelineRuns, 2)
	assert.Equal(t, uint(2), pipelineRuns[0].ID)
	assert.Equal(t, uint(1), pipelineRuns[1].ID)
}

func TestFailInterruptedPipelineRunsFailsWhenStorageFails(t *testing.T) {
	t.Run("finding running runs", func(t *testing.T) {
		pipelineRunApplication, pipelineRunRepository := newPipelineRunApplication(t)
		pipelineRunRepository.EXPECT().FindRunning(gomock.Any()).Return(nil, errors.New("disk"))

		_, failError := pipelineRunApplication.FailInterruptedPipelineRuns(context.Background())

		assert.ErrorContains(t, failError, "disk")
	})

	t.Run("updating a run", func(t *testing.T) {
		pipelineRunApplication, pipelineRunRepository := newPipelineRunApplication(t)
		pipelineRunRepository.EXPECT().FindRunning(gomock.Any()).Return([]entities.PipelineRun{{ID: 3}}, nil)
		pipelineRunRepository.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("disk"))

		_, failError := pipelineRunApplication.FailInterruptedPipelineRuns(context.Background())

		assert.ErrorContains(t, failError, "disk")
	})
}
