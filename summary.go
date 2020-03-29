package spidomtr

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	barChar   = "∎"
	checkMark = "√"
	crossMark = "☓"
	skipMark  = "-"
)

func showSummary(res Result) {
	// Total test summary
	fmt.Print("\nSummary:\n")
	fmt.Printf("%2s%-10s %d\n", "", "Count:", res.Stats.Count)
	fmt.Printf("%2s%-10s %s\n", "", "Total:", res.Stats.Duration)
	fmt.Printf("%2s%-10s %d ms\n", "", "Slowest:", int64(res.Stats.Slowest/time.Millisecond))
	fmt.Printf("%2s%-10s %d ms\n", "", "Fastest:", int64(res.Stats.Fastest/time.Millisecond))
	fmt.Printf("%2s%-10s %d ms\n", "", "Average:", int64(res.Stats.Average/time.Millisecond))
	fmt.Printf("%2s%-10s %4.2f\n", "", "Req/sec:", res.Stats.RPS)

	// Response time histogram
	fmt.Print("\nResponse time histogram:\n")
	fmt.Print(histogramStr(res.Stats.Histogram))

	// Latency distributions
	fmt.Print("\nLatency distribution:\n")
	for _, d := range res.Stats.Distributions {
		if d.Latency > 0 && d.Percentage > 0 {
			fmt.Printf("%2s%d%% in %d ms\n", "", d.Percentage, int64(d.Latency/time.Millisecond))
		}
	}

	// Responses
	fmt.Print("\nResponses:\n")
	fmt.Printf("%2s%-10s %d\n", "", "OK:", res.Stats.Passed)
	fmt.Printf("%2s%-10s %d\n", "", "Errored:", res.Stats.Errors)
	fmt.Printf("%2s%-10s %d\n", "", "Skipped:", res.Stats.Skips)

	// Print error distribution
	if len(res.Stats.Errorm) > 0 {
		keys := make([]string, 0, len(res.Stats.Errorm))
		for k := range res.Stats.Errorm {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fmt.Print("\nError distribution:\n")
		for _, err := range keys {
			fmt.Printf("%2s[%v] %s\n", "", res.Stats.Errorm[err], err)
		}
	}

	// Print stats on each test
	fmt.Print("\nTests:\n")
	for k, testStats := range res.TestStats {
		fmt.Print("\n")
		fmt.Printf("%2s%-10s\n", "", toMark(testStats)+" "+k)
		fmt.Printf("%4s%-10s %v\n", "", "Count:", testStats.Stats.Count)
		fmt.Printf("%4s%-10s %v\n", "", "OK:", testStats.Stats.Passed)
		fmt.Printf("%4s%-10s %v\n", "", "Errored:", testStats.Stats.Errors)
		fmt.Printf("%4s%-10s %v\n", "", "Skipped:", testStats.Stats.Skips)
		if testStats.Stats.Slowest > 0 {
			fmt.Printf("%4s%-10s %v ms\n", "", "Slowest:", int64(testStats.Stats.Slowest/time.Millisecond))
		}
		if testStats.Stats.Fastest > 0 {
			fmt.Printf("%4s%-10s %v ms\n", "", "Fastest:", int64(testStats.Stats.Fastest/time.Millisecond))
		}
		if testStats.Stats.Average > 0 {
			fmt.Printf("%4s%-10s %v ms\n", "", "Average:", int64(testStats.Stats.Average/time.Millisecond))
		}

		for _, d := range testStats.Stats.Distributions {
			if d.Percentage >= 90 {
				strlatency := strconv.FormatInt(int64(d.Latency/time.Millisecond), 10)
				fmt.Printf("%4s%-10s %s ms\n", "", strconv.Itoa(d.Percentage)+"%:", strlatency)
			}
		}

		if testStats.Stats.RPS > 0 {
			fmt.Printf("%4s%-10s %4.2f\n", "", "Req/sec:", testStats.Stats.RPS)
		}

		// Print error distribution
		if len(testStats.Stats.Errorm) > 0 {
			fmt.Printf("%4s%-10s\n", "", "Errors:")
			for err, count := range testStats.Stats.Errorm {
				fmt.Printf("%8s[%v] %s\n", "", count, err)
			}
		}
	}
}

func histogramStr(buckets []Bucket) string {
	max := 0
	for _, b := range buckets {
		if v := b.Count; v > max {
			max = v
		}
	}
	res := new(bytes.Buffer)
	for _, b := range buckets {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = (b.Count*40 + max/2) / max
		}
		if b.Count > 0 {
			res.WriteString(fmt.Sprintf("%2s%4s ms %-8s%s%s\n", "", strconv.FormatInt(int64(b.Mark/time.Millisecond), 10),
				"["+strconv.Itoa(b.Count)+"]", "|", strings.Repeat(barChar, barLen)))
		}
	}
	return res.String()
}

func toMark(stats TestStats) string {
	if stats.Stats.Errors > 0 {
		return crossMark
	}
	if stats.Stats.Skips > 0 {
		return skipMark
	}
	if stats.Stats.Passed > 0 {
		return checkMark
	}
	return ""
}
