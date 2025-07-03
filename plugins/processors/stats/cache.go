package stats

import (
	"sync"

	"github.com/gekatateam/neptunus/core"
)

type cache interface {
	observe(m *metric, v float64)
	flush(out chan<- *core.Event, flushFn func(m *metric, ch chan<- *core.Event))
	clear()
}

type individualCache map[uint64]*metric

func newIndividualCache() individualCache {
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

func (c individualCache) flush(out chan<- *core.Event, flushFn func(m *metric, ch chan<- *core.Event)) {
	for _, m := range c {
		flushFn(m, out)
		m.reset()
	}
}

func (c individualCache) clear() {
	clear(c)
}

var ss = &sharedStorage{
	s:  make(map[uint64]*sharedCache),
	mu: &sync.Mutex{},
}

type sharedStorage struct {
	s  map[uint64]*sharedCache
	mu *sync.Mutex
}

func (s *sharedStorage) delete(k uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.s, k)
}

func (s *sharedStorage) newCache(k uint64) *sharedCache {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cache, ok := s.s[k]; ok {
		cache.writers++
		return cache
	}

	cache := &sharedCache{
		id:        k,
		writers:   1,
		cache:     newIndividualCache(),
		syncPoint: 0,
		mu:        &sync.Mutex{},
	}
	s.s[k] = cache

	return cache
}

func newSharedCache(k uint64) *sharedCache {
	return ss.newCache(k)
}

type sharedCache struct {
	id      uint64
	writers int64

	cache individualCache

	syncPoint int64
	mu        *sync.Mutex
}

func (c *sharedCache) observe(m *metric, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.observe(m, v)
}

func (c *sharedCache) flush(out chan<- *core.Event, flushFn func(m *metric, ch chan<- *core.Event)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.syncPoint++
	if c.syncPoint == c.writers {
		for _, m := range c.cache {
			flushFn(m, out)
			m.reset()
		}
		c.syncPoint = 0
	}
}

func (c *sharedCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.syncPoint == 0 {
		c.cache.clear()
		ss.delete(c.id)
	}
}
