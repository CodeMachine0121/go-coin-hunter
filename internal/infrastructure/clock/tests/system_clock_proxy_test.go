package clock_test

import (
	"testing"
	"time"

	"github.com/CodeMachine0121/go-coin-hunter/internal/infrastructure/clock"
	"github.com/stretchr/testify/assert"
)

func TestSystemClockProxyReadsUniversalTime(t *testing.T) {
	before := time.Now()

	now := clock.NewSystemClockProxy().Now()

	assert.Equal(t, time.UTC, now.Location())
	assert.False(t, now.Before(before.Truncate(time.Second)))
}
