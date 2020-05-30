package util

import "time"

type StopWatch struct {
	start time.Time
}

// Return an auto-start StopWatch.
func NewStopWatch() *StopWatch {
	return &StopWatch{start: time.Now()}
}

// Duration will reset the StopWatch.
func (sw *StopWatch) Duration(reset bool) (d time.Duration) {
	now := time.Now()
	d = now.Sub(sw.start)
	if reset {
		sw.start = now
	}
	return
}
