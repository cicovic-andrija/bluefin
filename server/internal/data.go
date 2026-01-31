package internal

import (
	"sync/atomic"

	"src.acicovic.me/divelog/server"
)

var _data_ptr_latest atomic.Pointer[server.DiveLog]

func AquireAccess() *server.DiveLog {
	return _data_ptr_latest.Load()
}

func SwapData(latest *server.DiveLog) {
	_data_ptr_latest.Store(latest)
}
