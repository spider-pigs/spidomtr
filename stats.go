package spidomtr

import "time"

// Bucket type
type Bucket struct {
	Count     int
	Frequency int
	Mark      time.Duration
}

// LatencyDist type
type LatencyDist struct {
	Percentage int
	Latency    time.Duration
}

func avgDuration(duration time.Duration, total int) time.Duration {
	if total == 0 {
		return 0
	}
	return duration / time.Duration(total)
}

func distributions(percentiles []int, latencies []time.Duration) []LatencyDist {
	data := make([]time.Duration, len(percentiles))
	for i, j := 0, 0; i < len(latencies) && j < len(percentiles); i++ {
		current := i * 100 / len(latencies)

		if current >= percentiles[j] {
			data[j] = latencies[i]
			j++
		}
	}

	res := make([]LatencyDist, len(percentiles))
	for i := 0; i < len(percentiles); i++ {
		if data[i] > 0 {
			lat := data[i]
			res[i] = LatencyDist{Percentage: percentiles[i], Latency: lat}
		}
	}
	return res
}

func histogram(resolution int, latencies []time.Duration, slowest, fastest time.Duration) []Bucket {
	bc := int64(resolution)
	buckets := make([]time.Duration, bc+1)
	counts := make([]int, bc+1)
	bs := int64(slowest-fastest) / bc
	for i := int64(0); i < bc; i++ {
		buckets[i] = time.Duration(int64(fastest) + int64(bs)*i)
	}
	buckets[bc] = slowest
	var bi int
	var max int
	for i := 0; i < len(latencies); {
		if latencies[i] <= buckets[bi] {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		}
	}
	res := make([]Bucket, len(buckets))
	latencyCount := len(latencies)
	if latencyCount > 0 {
		for i := 0; i < len(buckets); i++ {
			res[i] = Bucket{
				Mark:      buckets[i],
				Count:     counts[i],
				Frequency: counts[i] / latencyCount,
			}
		}
	}

	return res
}
