package metrics

import "time"

func floatSeconds(begin time.Time) float64 {
	return time.Since(begin).Seconds()
}
