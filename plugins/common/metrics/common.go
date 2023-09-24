package metrics

import "time"

func floatSeconds(begin time.Time) float64 {
	return float64(time.Since(begin)) / float64(time.Second)
}
