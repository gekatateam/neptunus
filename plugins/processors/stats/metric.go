package stats

import "hash/fnv"

type metric struct {
	Descr metricDescr
	Value metricValue
	Stats metricStats
}

func (m metric) hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.Descr.Name))
	for _, v := range m.Descr.Labels {
		h.Write([]byte(v.Key))
		h.Write([]byte(v.Value))
	}
	return h.Sum64()
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
	Count float64
	Sum   float64
	Gauge float64
	Ang   float64
	Min   float64
	Max   float64
}

type metricStats struct {
	Count bool
	Sum   bool
	Gauge bool
	Ang   bool
	Min   bool
	Max   bool
}
