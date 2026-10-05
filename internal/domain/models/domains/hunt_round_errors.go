package domains

import "errors"

// ErrHuntRoundAlreadyRunning keeps rounds from overlapping: two rounds would read each other's upstream runs.
var ErrHuntRoundAlreadyRunning = errors.New("已有獵捕回合進行中")
