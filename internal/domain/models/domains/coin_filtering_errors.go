package domains

import "errors"

var (
	ErrNoSucceededDiscoveryRun = errors.New("尚無成功的探索輪次")
	// ErrCoinProfileSourceUnavailable means a whole data source failed, so no candidate can be judged fairly.
	ErrCoinProfileSourceUnavailable = errors.New("資料來源無法取得")
)
