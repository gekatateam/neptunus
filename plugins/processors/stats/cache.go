package stats

import "sync"

type cache struct {
	c  map[uint64]*metric
	mu *sync.Mutex
}

func newIndividualCache() *cache {
	return &cache{
		c:  make(map[uint64]*metric),
		mu: &sync.Mutex{},
	}
}

func (c *cache) observe(m *metric, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if metric, ok := c.c[m.hash()]; ok {
		m = metric
	} else { // hit an uncached netric
		c.c[m.hash()] = m
	}

	m.observe(v)
}

func (c *cache) flush(flushFn func(m *metric)) {
	for _, m := range c.c {
		c.mu.Lock()
		flushFn(m)
		c.mu.Unlock()
	}
}

func (c *cache) clear() {
	clear(c.c)
}
