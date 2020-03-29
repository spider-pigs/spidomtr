package runner

import "time"

// Timer type
type Timer struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration
}

// NewTimer creates timer
func NewTimer() *Timer {
	return &Timer{}
}

// Begin starts timer
func (timer *Timer) Begin() {
	timer.Start = time.Now()
}

// Finish stops timer
func (timer *Timer) Finish() {
	timer.End = time.Now()
	timer.Duration = timer.End.Sub(timer.Start)
}
