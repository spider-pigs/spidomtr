package handlers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spider-pigs/spidomtr"
)

const barWidth = 68

type progressBar struct {
	count   int
	current int
	start   time.Time
}

// ProgressBar is a runner handler that displays a running progress
// bar.
func ProgressBar() spidomtr.RunnerHandler {
	return &progressBar{}
}

// RunnerStarted is called when runner is started (prior to any tests
// have been run).
func (b *progressBar) RunnerStarted(id, description string, count int) {
	b.count = count
	b.start = time.Now()
}

// TestDone is called when a test has been completed.
func (b *progressBar) TestDone(spidomtr.TestResult) {
	b.current++
	filled := barWidth * b.current / b.count
	var bar string
	if b.current == b.count {
		bar = strings.Repeat("=", barWidth)
	} else if filled > 0 {
		bar = strings.Repeat("=", filled-1) + ">" + strings.Repeat(" ", barWidth-filled)
	} else {
		bar = strings.Repeat(" ", barWidth)
	}
	pct := 100 * b.current / b.count
	elapsed := time.Since(b.start).Truncate(time.Second)
	fmt.Fprintf(os.Stderr, "\r[%s] %3d%% %4s", bar, pct, elapsed)
}

// RunnerDone is called when the runner has run all tests.
func (b *progressBar) RunnerDone(spidomtr.Result) {
	fmt.Fprintln(os.Stderr)
}
