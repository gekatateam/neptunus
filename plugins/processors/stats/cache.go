package stats

import (
	"maps"
	"sync"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/refreshmap"
)

type cache interface {
	Size() int

	observe(m *metric, b map[float64]float64, v float64)
	flush(out chan<- *core.Event, flushFn func(m *metric, ch chan<- *core.Event))
	dropOlderThan(olderThan time.Duration)
	clear()
}

type storedMetric struct {
	m *metric
	d time.Time
}

type individualCache struct {
	c map[uint64]storedMetric
}

func newIndividualCache() individualCache {
	return individualCache{
		c: make(map[uint64]storedMetric),
	}
}

func (c individualCache) observe(m *metric, b map[float64]float64, v float64) {
	hash := m.hash()

	if stored, ok := c.c[hash]; ok {
		m = stored.m
	} else {
		// hit an uncached metric
		// add buckets into it, once
		m.Value.Buckets = maps.Clone(b)
	}

	c.c[hash] = storedMetric{
		m: m,
		d: time.Now(),
	}

	m.observe(v)
}

func (c individualCache) flush(out chan<- *core.Event, flushFn func(m *metric, ch chan<- *core.Event)) {
	for _, stored := range c.c {
		flushFn(stored.m, out)
		stored.m.reset()
	}
}

func (c individualCache) dropOlderThan(olderThan time.Duration) {
	prev := len(c.c)

	for k, v := range c.c {
		if time.Since(v.d) > olderThan {
			delete(c.c, k)
		}
	}

	c.c = refreshmap.RefreshIfNeeded(c.c, prev)
}

func (c individualCache) clear() {
	clear(c.c)
}

func (c individualCache) Size() int {
	return len(c.c)
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

type sharedCache struct {
	id      uint64
	writers int64

	cache individualCache

	syncPoint int64
	mu        *sync.Mutex
}

func newSharedCache(k uint64) *sharedCache {
	return ss.newCache(k)
}

func (c *sharedCache) observe(m *metric, b map[float64]float64, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.observe(m, b, v)
}

func (c *sharedCache) flush(out chan<- *core.Event, flushFn func(m *metric, ch chan<- *core.Event)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.syncPoint++
	if c.syncPoint == c.writers {
		c.cache.flush(out, flushFn)
		c.syncPoint = 0
	}
}

func (c *sharedCache) dropOlderThan(olderThan time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.syncPoint == 0 {
		c.cache.dropOlderThan(olderThan)
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

func (c *sharedCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cache.Size()
}
