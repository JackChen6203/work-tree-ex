package httpserver

import (
	"context"
	"sync"
)

var (
	readinessMu    sync.RWMutex
	readinessProbe func(context.Context) error
)

func SetReadinessProbe(probe func(context.Context) error) {
	readinessMu.Lock()
	defer readinessMu.Unlock()
	readinessProbe = probe
}

func getReadinessProbe() func(context.Context) error {
	readinessMu.RLock()
	defer readinessMu.RUnlock()
	return readinessProbe
}
