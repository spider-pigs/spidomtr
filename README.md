# spidomtr
[![Build Status](https://travis-ci.org/spider-pigs/spidomtr.svg?branch=master)](https://travis-ci.org/spider-pigs/spidomtr) [![Go Report Card](https://goreportcard.com/badge/github.com/spider-pigs/spidomtr)](https://goreportcard.com/report/github.com/spider-pigs/spidomtr) [![GoDoc](https://godoc.org/github.com/spider-pigs/spidomtr?status.svg)](https://godoc.org/github.com/spider-pigs/spidomtr)

spidomtr is a golang lib for benchmarking and load testing.

```console
               .__    .___              __
  ____________ |__| __| _/____   ______/  |________
 /  ___/\____ \|  |/ __ |/  _ \ /     \   __\_  __ \
 \___ \ |  |_> >  / /_/ (  <_> )  Y Y  \  |  |  | \/
/____  >|   __/|__\____ |\____/|__|_|  /__|  |__|
     \/ |__|           \/            \/

[====================================================================] 100%    3s

Summary:
  Count:     500
  Total:     3.391898304s
  Slowest:   104 ms
  Fastest:   21 ms
  Average:   61 ms
  Req/sec:   147.41

Response time histogram:
    21 ms [1]     |
    37 ms [94]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    54 ms [120]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    70 ms [95]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    87 ms [104]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
   104 ms [86]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎

Latency distribution:
  10% in 30 ms
  25% in 42 ms
  50% in 59 ms
  75% in 81 ms
  90% in 93 ms
  95% in 97 ms
  99% in 101 ms

Responses:
  OK:        500
  Errored:   0
  Skipped:   0

Tests:
  √ awesome_test
    Count:     500
    OK:        500
    Errored:   0
    Skipped:   0
    Slowest:   104 ms
    Fastest:   21 ms
    Average:   61 ms
    90%:       93 ms
    95%:       97 ms
    99%:       101 ms
    Req/sec:   147.41
```

# Install
```golang
import "github.com/spider-pigs/spidomtr"
```

# Usage
```golang
package main

import (
	"context"

	"github.com/spider-pigs/spidomtr"
	"github.com/spider-pigs/spidomtr/pkg/handlers"
	"github.com/spider-pigs/spidomtr/pkg/testunit"
)

func main() {
	// Create the test runner
	runner := spidomtr.NewRunner(
		spidomtr.ID("awesome tests"),
		spidomtr.Description("just running some awesome tests"),
		spidomtr.Handlers(
			handlers.ProgressBar(), // Displays progress bar during test run
		),
		spidomtr.Iterations(50), // Run the test 50 times
		spidomtr.Timeout(time.Second*20), // Set a test timeout
		spidomtr.Users(10), // Simulate 10 concurrent users
	)

	// Create a test
	test := testunit.New(
		testunit.Test(func(ctx context.Context, args []interface{}) ([]interface{}, error) {
			if err := doSomethingCool(ctx); err != nil {
				// Test failed
				return args, err
			}
			// Test passed
			return args, nil
		}),
	)

	// Run the test
	res := runner.Run(context.Background(), test)
	...
}
```
