package api

import (
	"runtime"
	"sync/atomic"
	"time"
)

const (
	StatusUpdateInterval = 60 // seconds
)

type Stats struct {
	NumGoroutine int    `json:"numGoroutine"`
	NumGC        uint32 `json:"numGC"`
	MemAllocMB   uint64 `json:"memAllocMB"`
	MemSysMB     uint64 `json:"memSysMB"`
}

type DebugAPI interface {
	GetStats() Stats
}

func newDebugAPI() DebugAPI {
	return &debugAPI{}
}

type debugAPI struct {
	lastUpdateTime int64
	stats          Stats
}

func (api *debugAPI) GetStats() Stats {
	now := time.Now().Unix()
	lastUpdateTime := atomic.LoadInt64(&api.lastUpdateTime)
	if now > lastUpdateTime+StatusUpdateInterval {
		if atomic.CompareAndSwapInt64(&api.lastUpdateTime, lastUpdateTime, now) {
			api.updateStats()
		}
	}

	return api.stats
}

func (api *debugAPI) updateStats() {
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	api.stats.NumGoroutine = runtime.NumGoroutine()
	api.stats.NumGC = memStats.NumGC
	api.stats.MemAllocMB = memStats.Alloc / 1024 / 1024
	api.stats.MemSysMB = memStats.Sys / 1024 / 1024
}
