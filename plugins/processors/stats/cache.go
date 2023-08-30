package stats

type cache interface {
	observe(m *metric, v float64)
	flush(flushFn func(m *metric))
	clear()
}

type individualCache map[uint64]*metric

func newIndividualCache() cache {
	return make(individualCache)
}

func (c individualCache) observe(m *metric, v float64) {
	if metric, ok := c[m.hash()]; ok {
		m = metric
	} else { // hit an uncached netric
		c[m.hash()] = m
	}

	m.observe(v)
}

func (c individualCache) flush(flushFn func(m *metric)) {
	for _, m := range c {
		flushFn(m)
		m.reset()
	}
}

func (c individualCache) clear() {
	clear(c)
}
