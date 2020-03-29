package runner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/spider-pigs/spidomtr/pkg/testunit"
)

// TestUnitDone type
type TestUnitDone func(testunit.TestUnit, *Timer, error)

// Runner type
type Runner struct {
	Iterations   int
	TestUnitDone TestUnitDone
	Timeout      time.Duration
}

// New constructs a new
func New(timeout time.Duration, iterations int) *Runner {
	return &Runner{
		Iterations: iterations,
		Timeout:    timeout,
	}
}

// Run runs test units.
func (runner Runner) Run(ctx context.Context, tests ...testunit.TestUnit) *Timer {
	totalTimer := NewTimer()
	totalTimer.Begin()

	for i := 0; i < runner.Iterations; i++ {
		for _, t := range tests {
			var err error
			timer := NewTimer()

			enabled, _ := t.Enabled()
			if enabled {
				timer, err = runTestUnit(ctx, t, runner.Timeout)
			}
			if runner.TestUnitDone != nil {
				runner.TestUnitDone(t, timer, err)
			}
		}
	}

	totalTimer.Finish()
	return totalTimer
}

func runTestUnit(ctx context.Context, t testunit.TestUnit, timeout time.Duration) (*Timer, error) {
	timer := NewTimer()
	var err error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				panicstr := fmt.Sprintf("%s", r)
				err = errors.New("func panic: " + panicstr)
			}
		}()
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Run prepare func
		var args []interface{}
		args, err = t.Prepare(ctx)
		if err != nil {
			return
		}

		// Run main func
		timer.Begin()
		args, err = t.Test(ctx, args)
		timer.Finish()

		if err != nil {
			return
		}

		// Run cleanup func
		err = t.Cleanup(ctx, args)
	}()
	wg.Wait()
	if err != nil {
		return NewTimer(), err
	}

	return timer, nil
}
