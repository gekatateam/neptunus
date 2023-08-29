package stats

import (
	"hash/fnv"
	"math"
)

var statsMap = map[string]bool{
	"count": true,
	"sum":   true,
	"gauge": true,
	"avg":   true,
	"min":   true,
	"max":   true,
}

type metric struct {
	Descr    metricDescr
	Value    metricValue
	Stats    metricStats
	Observed bool
}

func (m *metric) hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.Descr.Name))
	for _, v := range m.Descr.Labels {
		h.Write([]byte(v.Key))
		h.Write([]byte(v.Value))
	}
	return h.Sum64()
}

func (m *metric) observe(value float64) {
	// count calc
	if m.Value.Count+1 == math.MaxFloat64 {
		m.Value.Count = 0
	}
	m.Value.Count += 1

	// sum calc
	if m.Value.Sum+value == math.MaxFloat64 {
		m.Value.Sum = 0
	}
	m.Value.Sum += value

	// save gauge
	m.Value.Gauge = value

	// avg, min and max calc
	if m.Observed {
		if value < m.Value.Min {
			m.Value.Min = value
		}

		if value > m.Value.Max {
			m.Value.Max = value
		}

		m.Value.count2 += 1
		// new average = old average * (n-1)/n + new value /n
		m.Value.Avg = m.Value.Avg*(m.Value.count2-1)/m.Value.count2 + value/m.Value.count2
	} else {
		m.Value.Avg = value
		m.Value.Min = value
		m.Value.Max = value
		m.Value.count2 = 1
		m.Observed = true
	}
}

// count and sum are not reset
func (m *metric) reset() {
	m.Observed = false
	m.Value.Gauge = 0
	m.Value.count2 = 0
	m.Value.Avg = 0
	m.Value.Min = 0
	m.Value.Max = 0
}

type metricDescr struct {
	Name   string
	Labels []metricLabel
}

type metricLabel struct {
	Key   string
	Value string
}

type metricValue struct {
	Count  float64
	Sum    float64
	Gauge  float64
	Avg    float64
	Min    float64
	Max    float64
	count2 float64 // count for moving average
}

type metricStats struct {
	Count bool
	Sum   bool
	Gauge bool
	Avg   bool
	Min   bool
	Max   bool
}

func stats(stats []string) metricStats {
	s := metricStats{}
	for _, v := range stats {
		switch v {
		case "sum":
			s.Sum = true
		case "count":
			s.Count = true
		case "gauge":
			s.Gauge = true
		case "avg":
			s.Avg = true
		case "min":
			s.Min = true
		case "max":
			s.Max = true
		}
	}
	return s
}
