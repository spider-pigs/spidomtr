# spidomtr
[![CI](https://github.com/spider-pigs/spidomtr/actions/workflows/ci.yml/badge.svg)](https://github.com/spider-pigs/spidomtr/actions/workflows/ci.yml) [![GoDoc](https://godoc.org/github.com/spider-pigs/spidomtr?status.svg)](https://godoc.org/github.com/spider-pigs/spidomtr)

spidomtr is a golang lib for benchmarking and load testing.

```console
               .__    .___              __
  ____________ |__| __| _/____   ______/  |________
 /  ___/\____ \|  |/ __ |/  _ \ /     \   __\_  __ \
 \___ \ |  |_> >  / /_/ (  <_> )  Y Y  \  |  |  | \/
/____  >|   __/|__\____ |\____/|__|_|  /__|  |__|
     \/ |__|           \/            \/

[====================================================================] 100%    4s

Summary:
  Count:     1500
  Total:     4.348195333s
  Slowest:   49 ms
  Fastest:   5 ms
  Average:   22 ms
  Req/sec:   229.98

Response time histogram:
     5 ms [1]     |∎
     6 ms [44]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    13 ms [52]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    18 ms [60]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    30 ms [12]    |∎∎∎∎∎∎∎∎
    43 ms [24]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    49 ms [10]    |∎∎∎∎∎∎∎

Latency distribution:
  10% in 7 ms
  25% in 11 ms
  50% in 17 ms
  75% in 29 ms
  90% in 41 ms
  95% in 45 ms
  99% in 49 ms

Responses:
  OK:        949
  Errored:   51
  Skipped:   500

Tests:
  ☓ flaky_endpoint
  - disabled_endpoint
  √ fast_endpoint
```

# Install
```golang
import "github.com/spider-pigs/spidomtr"
```

# Usage

See [example/main.go](example/main.go) for a complete runnable example.

```golang
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
```
