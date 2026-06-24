package handlers

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/spider-pigs/spidomtr"
	"github.com/spider-pigs/spidomtr/pkg/testunit"
)

func colorize(color, s string) string {
	return "\033[" + color + "m" + s + "\033[0m"
}

func yellow(s string) string { return colorize("33", s) }
func red(s string) string    { return colorize("31", s) }
func green(s string) string  { return colorize("32", s) }

const (
	checkMark = "√"
	crossMark = "☓"
	skipMark  = "-"
)

// TestLogger type
type TestLogger struct {
	Log    *log.Logger
	Buffer *bytes.Buffer
}

// Logger logs tests during the test run
func Logger() spidomtr.RunnerHandler {
	return &TestLogger{}
}

// RunnerStarted is called when runner is started (prior to any tests
// have been run).
func (logger *TestLogger) RunnerStarted(id, description string, count int) {
	logger.Log = log.New(os.Stdout, "", 0)
	logger.Buffer = &bytes.Buffer{}

	fmt.Fprintf(logger.Buffer, "Running %s: %s...\n", id, description)
}

// TestDone is called when a test has been completed.
func (logger *TestLogger) TestDone(res spidomtr.TestResult) {
	switch res.Outcome {
	case testunit.Skip:
		fmt.Fprintf(logger.Buffer, "%s %s: %s\n", skipMark, res.ID, res.Comment)
	case testunit.Fail:
		fmt.Fprintf(logger.Buffer, "%s %s: %s\n", crossMark, res.ID, res.Error)
	case testunit.Pass:
		fmt.Fprintf(logger.Buffer, "%s %s: %s\n", checkMark, res.ID, res.Duration)
	}
}

// RunnerDone is called when the runner has run all tests.
func (logger *TestLogger) RunnerDone(res spidomtr.Result) {
	mark := func() string {
		switch {
		case res.Stats.Errors > 0:
			return red(crossMark)
		case res.Stats.Skips > 0:
			return yellow(skipMark)
		}
		return green(checkMark)
	}()

	str := fmt.Sprintf("%d/%d passed, %d failed, %d skipped, took %s, avg %s/test", res.Stats.Passed, res.Stats.Count, res.Stats.Errors, res.Stats.Skips, res.Stats.Duration, res.Stats.Average)
	divider := strings.Repeat("-", utf8.RuneCountInString(str)+2)

	fmt.Fprintf(logger.Buffer, "%s\n%s %s\n%s\n", divider, mark, str, divider)
	logger.Log.Print(logger.Buffer.String())
}
