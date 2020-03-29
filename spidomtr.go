package spidomtr

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/spider-pigs/spidomtr/internal/runner"
	"github.com/spider-pigs/spidomtr/pkg/testunit"
)

const asciilogo = `
               .__    .___              __
  ____________ |__| __| _/____   ______/  |________
 /  ___/\____ \|  |/ __ |/  _ \ /     \   __\_  __ \
 \___ \ |  |_> >  / /_/ (  <_> )  Y Y  \  |  |  | \/
/____  >|   __/|__\____ |\____/|__|_|  /__|  |__|
     \/ |__|           \/            \/             `

// DefaultHistogramBuckets is the default number of buckets
const DefaultHistogramBuckets = 40

// DefaultPercentiles the default percentiles
var DefaultPercentiles = []int{10, 25, 50, 75, 90, 95, 99}

// Result type
type Result struct {
	ChildResults []Result
	Date         time.Time
	Stats        Stats
	TestStats    map[string]TestStats
}

// TestResult type
type TestResult struct {
	Comment  string
	Date     time.Time
	Duration time.Duration
	End      time.Time
	Error    error
	ID       string
	Outcome  testunit.TestOutcome
	Start    time.Time
}

// Stats type
type Stats struct {
	Average       time.Duration
	Count         int
	Description   string
	Distributions []LatencyDist
	Duration      time.Duration
	Durations     []time.Duration
	End           time.Time
	Errorm        map[string]int
	Errors        int
	Fastest       time.Duration
	Histogram     []Bucket
	Passed        int
	RPS           float64
	Skips         int
	Slowest       time.Duration
	Start         time.Time
}

// TestStats type
type TestStats struct {
	TestResults []TestResult
	Stats       Stats
}

// Config type
type Config struct {
	Description      string
	ID               string
	Iterations       int
	Handlers         []RunnerHandler
	HistogramBuckets int
	Percentiles      []int
	ShowLogo         bool
	ShowSummary      bool
	Timeout          time.Duration
	Users            int
}

// Option type
type Option func(*Config)

// Description sets a description
func Description(description string) Option {
	return func(cfg *Config) {
		cfg.Description = description
	}
}

// Handlers sets runner handlers
func Handlers(h ...RunnerHandler) Option {
	return func(cfg *Config) {
		cfg.Handlers = h
	}
}

// HistogramBuckets sets histogram resolution (defaults to 40 buckets)
func HistogramBuckets(buckets int) Option {
	return func(cfg *Config) {
		cfg.HistogramBuckets = buckets
	}
}

// ID sets ID
func ID(s string) Option {
	return func(cfg *Config) {
		cfg.ID = s
	}
}

// Iterations sets number of iterations
func Iterations(i int) Option {
	return func(cfg *Config) {
		cfg.Iterations = i
	}
}

// ShowLogo should the logo be displayed (defaults to true)
func ShowLogo(b bool) Option {
	return func(cfg *Config) {
		cfg.ShowLogo = b
	}
}

// ShowSummary should the summary be displayed (defaults to true)
func ShowSummary(b bool) Option {
	return func(cfg *Config) {
		cfg.ShowSummary = b
	}
}

// Percentiles sets percentiles for latency distributions
func Percentiles(p []int) Option {
	return func(cfg *Config) {
		cfg.Percentiles = p
	}
}

// Timeout sets test timout (defaults to 10 secs)
func Timeout(t time.Duration) Option {
	return func(cfg *Config) {
		cfg.Timeout = t
	}
}

// Users sets number of users
func Users(users int) Option {
	return func(cfg *Config) {
		cfg.Users = users
	}
}

// RunnerHandler interface
type RunnerHandler interface {
	// RunnerStarted is called when runner is started (prior to any
	// tests have been run).
	RunnerStarted(id, description string, testUnits int)
	// TestDone is called when a test has been completed.
	TestDone(res TestResult)
	// RunnerDone is called when the runner has run all tests.
	RunnerDone(res Result)
}

// Runner type
type Runner struct {
	cfg *Config
}

// NewRunner constructs a new runner
func NewRunner(options ...Option) *Runner {
	cfg := &Config{
		HistogramBuckets: DefaultHistogramBuckets,
		Iterations:       1,
		Percentiles:      DefaultPercentiles,
		ShowLogo:         true,
		ShowSummary:      true,
		Timeout:          10 * time.Second,
		Users:            1,
	}

	for _, opt := range options {
		opt(cfg)
	}

	return &Runner{cfg: cfg}
}

// Run runs tests
func (r *Runner) Run(ctx context.Context, tests ...testunit.TestUnit) Result {
	if r.cfg.ShowLogo {
		fmt.Printf("%s\n\n", asciilogo)
	}

	count := len(tests) * r.cfg.Iterations * r.cfg.Users

	for _, h := range r.cfg.Handlers {
		h.RunnerStarted(r.cfg.ID, r.cfg.Description, count)
	}

	if r.cfg.Users == 1 {
		res := r.run(ctx, tests...)
		for _, h := range r.cfg.Handlers {
			h.RunnerDone(res)
		}
		if r.cfg.ShowSummary {
			showSummary(res)
		}
		return res
	}

	var wg sync.WaitGroup
	wg.Add(r.cfg.Users)
	mux := &sync.Mutex{}

	results := make([]Result, 0)
	for i := 0; i < r.cfg.Users; i++ {
		go func() {
			defer wg.Done()
			runner := NewRunner()
			runner.cfg = r.cfg
			res := runner.run(ctx, tests...)
			mux.Lock()
			defer mux.Unlock()
			results = append(results, res)
		}()
	}
	wg.Wait()

	sum := JoinResults(r.cfg.HistogramBuckets, r.cfg.Percentiles, results...)

	// Notify handlers
	for _, h := range r.cfg.Handlers {
		h.RunnerDone(sum)
	}

	if r.cfg.ShowSummary {
		showSummary(sum)
	}

	return sum
}

// Run runs tests
func (r *Runner) run(ctx context.Context, tests ...testunit.TestUnit) Result {
	if hasDuplicateIDs(tests) {
		panic("tests have duplicate ids")
	}

	durations := make([]time.Duration, 0)

	var skipped, errored int
	testRunner := runner.New(r.cfg.Timeout, r.cfg.Iterations)

	errorm := make(map[string]int)
	testStats := make(map[string]TestStats)
	testRunner.TestUnitDone = func(t testunit.TestUnit, timer *runner.Timer, err error) {
		enabled, description := t.Enabled()

		// Set test outcome
		outcome := func() testunit.TestOutcome {
			if !enabled {
				return testunit.Skip
			} else if err != nil {
				return testunit.Fail
			}
			return testunit.Pass
		}()

		// Set result
		testResult := TestResult{
			Date:     time.Now(),
			Duration: timer.Duration,
			End:      timer.End,
			Error:    err,
			ID:       t.ID(),
			Outcome:  outcome,
			Start:    timer.Start,
		}

		// Handle the test outcome
		switch outcome {
		case testunit.Skip:
			skipped++
			testResult.Comment = description
		case testunit.Fail:
			errored++
			errorm[err.Error()]++
			testResult.Comment = err.Error()
		case testunit.Pass:
			durations = append(durations, timer.Duration)
		}

		// Append result to previous results
		v, ok := testStats[t.ID()]
		if !ok {
			testResults := make([]TestResult, 0)
			v = TestStats{
				TestResults: testResults,
			}
		}
		v.TestResults = append(v.TestResults, testResult)
		testStats[t.ID()] = v

		// Report test result to handlers.
		for _, h := range r.cfg.Handlers {
			h.TestDone(testResult)
		}
	}

	// Run the tests
	timer := testRunner.Run(ctx, tests...)

	// Gather stats
	count := len(tests) * r.cfg.Iterations
	passed := count - errored - skipped
	avg := avgDuration(timer.Duration, passed)

	// Sort durations in ascending order
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	var fastest, slowest time.Duration
	if len(durations) > 0 {
		fastest = durations[0]
		slowest = durations[len(durations)-1]
	}

	distributions := distributions(r.cfg.Percentiles, durations)
	histogram := histogram(r.cfg.HistogramBuckets, durations, slowest, fastest)

	rps := float64(0)
	if timer.Duration > 0 {
		rps = float64(passed+errored) / timer.Duration.Seconds()
	}

	stats := Stats{
		Average:       avg,
		Count:         count,
		Distributions: distributions,
		Durations:     durations,
		Duration:      timer.Duration,
		Errorm:        errorm,
		Errors:        errored,
		Fastest:       fastest,
		End:           timer.End,
		Histogram:     histogram,
		Passed:        passed,
		RPS:           rps,
		Skips:         skipped,
		Slowest:       slowest,
		Start:         timer.Start,
	}

	calcTestStats(r.cfg.HistogramBuckets, r.cfg.Percentiles, testStats)

	return Result{
		Date:      time.Now(),
		Stats:     stats,
		TestStats: testStats,
	}
}

func calcTestStats(buckets int, percentiles []int, testStats map[string]TestStats) {
	for k, v := range testStats {
		// Create stats from invidual tests
		durations := make([]time.Duration, 0)
		var accduration time.Duration
		starts := make([]time.Time, 0)
		ends := make([]time.Time, 0)
		errorm := make(map[string]int)
		ok, skips, err := 0, 0, 0
		for _, r := range v.TestResults {
			if r.Error != nil {
				errorm[r.Error.Error()]++
			}
			accduration += r.Duration
			if !r.Start.IsZero() {
				starts = append(starts, r.Start)
				ends = append(ends, r.End)
			}
			switch r.Outcome {
			case testunit.Pass:
				ok++
				durations = append(durations, r.Duration)
			case testunit.Fail:
				err++
			case testunit.Skip:
				skips++
			}
		}

		// Sort in starts asc
		sort.Slice(starts, func(i, j int) bool {
			return starts[i].Before(starts[j])
		})

		// Sort in ends desc
		sort.Slice(ends, func(i, j int) bool {
			return ends[i].After(ends[j])
		})

		// Sort in duration asc
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})

		var start time.Time
		var end time.Time
		if len(starts) > 0 {
			start = starts[0]
		}
		if len(ends) > 0 {
			end = ends[0]
		}
		duration := end.Sub(start)

		var fastest, slowest time.Duration
		if len(durations) > 0 {
			fastest = durations[0]
			slowest = durations[len(durations)-1]
		}

		hist := histogram(buckets, durations, slowest, fastest)
		dist := distributions(percentiles, durations)
		c := len(v.TestResults)
		v.Stats.Average = avgDuration(accduration, ok)
		v.Stats.Count = c
		v.Stats.Distributions = dist
		v.Stats.Duration = duration
		v.Stats.Durations = durations
		v.Stats.Errors = err
		v.Stats.Errorm = errorm
		v.Stats.Fastest = fastest
		v.Stats.Histogram = hist
		v.Stats.Passed = ok
		v.Stats.Skips = skips
		v.Stats.Slowest = slowest

		rps := float64(0)
		if duration > 0 {
			rps = float64(ok+err) / duration.Seconds()
		}

		v.Stats.RPS = rps
		testStats[k] = v
	}
}

// JoinResults joins results to a combined result
func JoinResults(histogramBuckets int, percentiles []int, results ...Result) Result {
	count, ok, skips, err := 0, 0, 0, 0
	var accduration time.Duration
	starts := make([]time.Time, 0)
	ends := make([]time.Time, 0)
	durations := make([]time.Duration, 0)
	errorm := make(map[string]int)
	for _, r := range results {
		count += r.Stats.Count
		accduration += r.Stats.Duration
		err += r.Stats.Errors
		for k, v := range r.Stats.Errorm {
			errorm[k] += v
		}
		if !r.Stats.Start.IsZero() {
			starts = append(starts, r.Stats.Start)
			ends = append(ends, r.Stats.End)
		}
		skips += r.Stats.Skips
		ok += r.Stats.Count - r.Stats.Skips - r.Stats.Errors
		durations = append(durations, r.Stats.Durations...)
	}

	// Sort in starts asc
	sort.Slice(starts, func(i, j int) bool {
		return starts[i].Before(starts[j])
	})

	// Sort in ends desc
	sort.Slice(ends, func(i, j int) bool {
		return ends[i].After(ends[j])
	})

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	start := starts[0]
	end := ends[0]
	totalDuration := end.Sub(start)

	avg := avgDuration(accduration, ok)
	dist := distributions(percentiles, durations)

	var fastest, slowest time.Duration
	if len(durations) > 0 {
		fastest = durations[0]
		slowest = durations[len(durations)-1]
	}

	hist := histogram(histogramBuckets, durations, slowest, fastest)

	testStats := func() map[string]TestStats {
		testStats := make(map[string]TestStats)
		for _, r := range results {
			for id, s := range r.TestStats {
				v, ok := testStats[id]
				if ok {
					v.TestResults = append(v.TestResults, s.TestResults...)
					testStats[id] = v
				} else {
					testStats[id] = s
				}
			}
		}
		return testStats
	}()

	// Collect all test stats
	calcTestStats(histogramBuckets, percentiles, testStats)

	rps := float64(0)
	if totalDuration > 0 {
		rps = float64(ok+err) / totalDuration.Seconds()
	}

	stats := Stats{
		Average:       avg,
		Count:         count,
		Distributions: dist,
		Duration:      totalDuration,
		Durations:     durations,
		End:           end,
		Errors:        err,
		Errorm:        errorm,
		Fastest:       fastest,
		Histogram:     hist,
		Passed:        ok,
		RPS:           rps,
		Skips:         skips,
		Slowest:       slowest,
		Start:         start,
	}

	res := Result{
		ChildResults: results,
		Date:         time.Now(),
		Stats:        stats,
		TestStats:    testStats,
	}

	return res
}

func hasDuplicateIDs(tests []testunit.TestUnit) bool {
	m := make(map[string]int)
	for _, t := range tests {
		_, ok := m[t.ID()]
		if ok {
			return true
		}
		m[t.ID()]++
	}
	return false
}
