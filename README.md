# spidomtr

[![CI](https://github.com/spider-pigs/spidomtr/actions/workflows/ci.yml/badge.svg)](https://github.com/spider-pigs/spidomtr/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/spider-pigs/spidomtr.svg)](https://pkg.go.dev/github.com/spider-pigs/spidomtr) [![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Load testing and benchmarking — from the command line or from Go.**

```text
               .__    .___              __
  ____________ |__| __| _/____   ______/  |________
 /  ___/\____ \|  |/ __ |/  _ \ /     \   __\_  __ \
 \___ \ |  |_> >  / /_/ (  <_> )  Y Y  \  |  |  | \/
/____  >|   __/|__\____ |\____/|__|_|  /__|  |__|
     \/ |__|           \/            \/
```

spidomtr comes in two flavours:

- **`spidomtr` CLI** — point it at a URL and get latency percentiles, a response time histogram and an error breakdown. No code required.
- **Go library** — write your own test units (HTTP, gRPC, databases, anything) and run them with concurrent users, timeouts and pluggable result handlers.

Both print the same report. Here are the key parts:

```console
$ spidomtr -n 100 -c 10 http://localhost:8080/

Summary:
  Count:     1000
  Total:     3.124614208s
  Slowest:   50 ms
  Fastest:   10 ms
  Average:   31 ms
  Req/sec:   320.04

Latency distribution:
  10% in 14 ms
  25% in 20 ms
  50% in 30 ms
  75% in 39 ms
  90% in 45 ms
  95% in 48 ms
  99% in 49 ms

Responses:
  OK:        950
  Errored:   50
  Skipped:   0

Error distribution:
  [50] unexpected status: 503 Service Unavailable
```

<details>
<summary>Full output (logo, progress bar, histogram and per-test stats)</summary>

```console
$ spidomtr -n 100 -c 10 -buckets 10 http://localhost:8080/
               .__    .___              __
  ____________ |__| __| _/____   ______/  |________
 /  ___/\____ \|  |/ __ |/  _ \ /     \   __\_  __ \
 \___ \ |  |_> >  / /_/ (  <_> )  Y Y  \  |  |  | \/
/____  >|   __/|__\____ |\____/|__|_|  /__|  |__|
     \/ |__|           \/            \/

[====================================================================] 100%   3s

Summary:
  Count:     1000
  Total:     3.124614208s
  Slowest:   50 ms
  Fastest:   10 ms
  Average:   31 ms
  Req/sec:   320.04

Response time histogram:
    10 ms [1]     |
    14 ms [91]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    18 ms [114]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    22 ms [96]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    26 ms [77]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    30 ms [85]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    34 ms [124]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    38 ms [101]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    42 ms [89]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    46 ms [89]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
    50 ms [83]    |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎

Latency distribution:
  10% in 14 ms
  25% in 20 ms
  50% in 30 ms
  75% in 39 ms
  90% in 45 ms
  95% in 48 ms
  99% in 49 ms

Responses:
  OK:        950
  Errored:   50
  Skipped:   0

Error distribution:
  [50] unexpected status: 503 Service Unavailable

Tests:

  ☓ GET http://localhost:8080/
    Count:     1000
    OK:        950
    Errored:   50
    Skipped:   0
    Slowest:   50 ms
    Fastest:   10 ms
    Average:   29 ms
    90%:       45 ms
    95%:       48 ms
    99%:       49 ms
    Req/sec:   320.05
    Errors:
        [50] unexpected status: 503 Service Unavailable
```

</details>

## Contents

- [CLI](#cli)
  - [Install](#install)
  - [Examples](#examples)
  - [Flags](#flags)
  - [Exit codes](#exit-codes)
- [Library](#library)
  - [Install](#install-1)
  - [Quick start](#quick-start)
  - [Test units](#test-units)
  - [Runner options](#runner-options)
  - [Handlers](#handlers)
- [License](#license)

## CLI

### Install

```console
go install github.com/spider-pigs/spidomtr/cmd/spidomtr@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`, then run `spidomtr -h` to see all flags.

### Examples

```console
# 10 concurrent users, 100 requests each (1000 requests in total)
spidomtr -n 100 -c 10 https://example.com

# POST a JSON body and only accept 201 Created
spidomtr -m POST -H 'Content-Type: application/json' -d '{"name":"spidomtr"}' -expect 201 https://example.com/api/users

# Read the request body from a file and send an auth header
spidomtr -m PUT -D payload.json -H "Authorization: Bearer $TOKEN" https://example.com/api/items/42

# Hit a staging server with a self-signed certificate and a custom Host header
spidomtr -k -H 'Host: app.internal' https://10.0.0.12/health

# Fail a CI job if any request errors or takes longer than 500 ms
spidomtr -n 50 -c 5 -t 500ms -no-logo -progress=false -fail-on-error https://example.com
```

Press <kbd>Ctrl</kbd>+<kbd>C</kbd> to stop a run early: in-flight requests are cancelled and the summary is still printed. Press it again to quit immediately.

### Flags

**Load**

| Flag | Default | Description |
|------|---------|-------------|
| `-n` | `10` | requests per user (total requests = `n × c`) |
| `-c` | `1` | concurrent users |
| `-t` | `10s` | timeout per request |

**Request**

| Flag | Default | Description |
|------|---------|-------------|
| `-m` | `GET` | HTTP method |
| `-H` | | request header `"Key: Value"`, repeatable. Requests send `User-Agent: spidomtr/<version>` unless you set your own (`-H 'User-Agent:'` sends none) |
| `-d` | | request body |
| `-D` | | read the request body from a file (cannot be combined with `-d`) |
| `-expect` | any status < 400 | comma-separated list of accepted status codes, e.g. `200,204` |
| `-k` | `false` | skip TLS certificate verification |
| `-disable-keepalive` | `false` | open a new connection for every request |
| `-no-redirects` | `false` | do not follow redirects (3xx responses then count as OK unless `-expect` says otherwise) |

**Output**

| Flag | Default | Description |
|------|---------|-------------|
| `-id` | the URL | id of the test run, shown with `-v` |
| `-desc` | | description of the test run, shown with `-v` |
| `-buckets` | `40` | number of histogram buckets |
| `-percentiles` | `10,25,50,75,90,95,99` | latency percentiles to report |
| `-progress` | `true` | show the progress bar (`-progress=false` to hide) |
| `-v` | `false` | print every request result after the run |
| `-no-logo` | `false` | hide the ASCII logo |
| `-fail-on-error` | `false` | exit with status `1` if any request failed |

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | run completed |
| `1` | at least one request failed and `-fail-on-error` was set |
| `2` | invalid flags or arguments |
| `130` | interrupted with <kbd>Ctrl</kbd>+<kbd>C</kbd> |

## Library

Use the library when you need more than a single URL: multiple endpoints, other protocols, setup and teardown per request, or custom reporting.

### Install

```console
go get github.com/spider-pigs/spidomtr
```

### Quick start

[example/main.go](example/main.go) is a complete, runnable example. It defines a fast test, a flaky test that fails now and then, and a skipped test, then runs them with 5 concurrent users:

```console
go run ./example
```

The example simulates its work with `time.Sleep`, so it runs without a server. To test something real, put your own code in `testunit.Test` (an HTTP call, a database query, anything) and return an error when it fails. The sections below describe the options it uses.

### Test units

A test unit is built with `testunit.New` and these options. Each iteration runs `Prepare` → `Test` → `Cleanup`, and **only `Test` is timed**.

| Option | Purpose |
|--------|---------|
| `testunit.ID(string)` | unique name shown in the report (random if omitted) |
| `testunit.Prepare(func(ctx) ([]interface{}, error))` | setup before each test; its return values are passed to `Test` |
| `testunit.Test(func(ctx, args) ([]interface{}, error))` | the code being measured; return an error to mark it failed |
| `testunit.Cleanup(func(ctx, args) error)` | teardown after a successful test |
| `testunit.Enabled(func() (bool, string))` | return `false` and a reason to skip the test |

You can also implement the `testunit.TestUnit` interface directly.

### Runner options

| Option | Default | Description |
|--------|---------|-------------|
| `spidomtr.ID(string)` | | id of the run |
| `spidomtr.Description(string)` | | description of the run |
| `spidomtr.Iterations(int)` | `1` | times each user runs every test |
| `spidomtr.Users(int)` | `1` | concurrent users |
| `spidomtr.Timeout(time.Duration)` | `10s` | deadline on the context passed to `Prepare`, `Test` and `Cleanup`; your code must respect `ctx` for it to take effect |
| `spidomtr.Handlers(...RunnerHandler)` | | handlers notified during the run |
| `spidomtr.HistogramBuckets(int)` | `40` | number of histogram buckets |
| `spidomtr.Percentiles([]int)` | `10,25,50,75,90,95,99` | latency percentiles to calculate |
| `spidomtr.ShowLogo(bool)` | `true` | print the ASCII logo |
| `spidomtr.ShowSummary(bool)` | `true` | print the summary report |

`Run` returns a `spidomtr.Result` with overall `Stats` and per-test `TestStats`, so you can assert on the numbers or export them yourself.

### Handlers

Handlers receive events while the runner works. Two are included in `pkg/handlers`:

| Handler | Description |
|---------|-------------|
| `handlers.ProgressBar()` | live progress bar on stderr |
| `handlers.Logger()` | prints every test result to stdout when the run finishes |

Write your own by implementing `spidomtr.RunnerHandler`. With more than one user, `TestDone` is called from several goroutines, so guard any shared state:

```go
type RunnerHandler interface {
	RunnerStarted(id, description string, testUnits int)
	TestDone(res TestResult)
	RunnerDone(res Result)
}
```

## License

[MIT](LICENSE)
