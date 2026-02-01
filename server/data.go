package server

import (
	"sync/atomic"
)

var _divelog_latest atomic.Pointer[DiveLog]

func acquireDataAccess() *DiveLog {
	return _divelog_latest.Load()
}

func swapLatestData(latest *DiveLog) {
	_divelog_latest.Store(latest)
}
