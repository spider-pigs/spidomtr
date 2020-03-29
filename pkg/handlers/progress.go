package handlers

import (
	"github.com/cheggaaa/pb/v3"
	"github.com/spider-pigs/spidomtr"
)

type progressBar struct {
	bar *pb.ProgressBar
}

// ProgressBar is a runner handler that displays a running progress
// bar.
func ProgressBar() spidomtr.RunnerHandler {
	return &progressBar{}
}

// RunnerStarted is called when runner is started (prior to any tests
// have been run).
func (b *progressBar) RunnerStarted(id, description string, count int) {
	b.bar = pb.StartNew(count)
}

// TestDone is called when a test has been completed.
func (b *progressBar) TestDone(spidomtr.TestResult) {
	b.bar.Increment()
}

// RunnerDone is called when the runner has run all tests.
func (b *progressBar) RunnerDone(spidomtr.Result) {
	b.bar.Finish()
}
