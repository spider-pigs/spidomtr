package main

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/spider-pigs/spidomtr"
	"github.com/spider-pigs/spidomtr/pkg/handlers"
	"github.com/spider-pigs/spidomtr/pkg/testunit"
)

func main() {
	// A simple test that simulates variable latency
	fast := testunit.New(
		testunit.ID("fast_endpoint"),
		testunit.Test(func(ctx context.Context, args []interface{}) ([]interface{}, error) {
			time.Sleep(time.Duration(5+rand.Intn(15)) * time.Millisecond)
			return nil, nil
		}),
	)

	// A test that occasionally fails
	flaky := testunit.New(
		testunit.ID("flaky_endpoint"),
		testunit.Test(func(ctx context.Context, args []interface{}) ([]interface{}, error) {
			time.Sleep(time.Duration(10+rand.Intn(40)) * time.Millisecond)
			if rand.Intn(10) == 0 {
				return nil, errors.New("connection timeout")
			}
			return nil, nil
		}),
	)

	// A skipped test
	skipped := testunit.New(
		testunit.ID("disabled_endpoint"),
		testunit.Enabled(func() (bool, string) {
			return false, "endpoint under maintenance"
		}),
	)

	runner := spidomtr.NewRunner(
		spidomtr.ID("example-run"),
		spidomtr.Description("demonstrate spidomtr features"),
		spidomtr.Iterations(100),
		spidomtr.Users(5),
		spidomtr.Timeout(10*time.Second),
		spidomtr.Handlers(handlers.ProgressBar(), handlers.Logger()),
	)

	runner.Run(context.Background(), fast, flaky, skipped)
}
