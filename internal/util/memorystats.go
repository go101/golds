package util

import (
	"fmt"
	"runtime"
)

func MemoryUse() string {
	var stats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&stats)
	if m := stats.Sys >> 20; m > 1000 {
		return fmt.Sprintf("%.1fG", float64(m)*0.001)
	} else if m > 10 {
		return fmt.Sprintf("%dM", m)
	} else {
		return fmt.Sprintf("%.1fM", float64(m))
	}
}
