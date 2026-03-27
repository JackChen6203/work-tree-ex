package cache

import "sync/atomic"

var distributedMode atomic.Bool

func SetDistributedMode(enabled bool) {
	distributedMode.Store(enabled)
}

func DistributedModeEnabled() bool {
	return distributedMode.Load()
}
