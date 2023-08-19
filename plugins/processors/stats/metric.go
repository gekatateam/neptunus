package stats

type metric struct {
	Descr metricDescr
	Value metricValue
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


