package stats

import "sync"

type individualCache struct {
	c  map[uint64]*metric
	mu *sync.Mutex
}

type cache interface {
	observe(m *metric, v float64)
	flush(flushFn func(m *metric))
	clear()
}

func newIndividualCache() cache {
	return &individualCache{
		c:  make(map[uint64]*metric),
		mu: &sync.Mutex{},
	}
}

func (c *individualCache) observe(m *metric, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if metric, ok := c.c[m.hash()]; ok {
		m = metric
	} else { // hit an uncached netric
		c.c[m.hash()] = m
	}

	m.observe(v)
}

func (c *individualCache) flush(flushFn func(m *metric)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, m := range c.c {
		flushFn(m)
		m.reset()
	}
}

func (c *individualCache) clear() {
	clear(c.c)
}
