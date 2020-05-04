package util

import "time"

type StopWatch struct {
	start time.Time
}

// Return an auto-start StopWatch.
func NewStopWatch() *StopWatch {
	return &StopWatch{start: time.Now()}
}

func (sw *StopWatch) Duration() (d time.Duration) {
	now := time.Now()
	d = now.Sub(sw.start)
	sw.start = now
	return
}
