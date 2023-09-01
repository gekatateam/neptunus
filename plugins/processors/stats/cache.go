package stats

import (
	"sync"
	"sync/atomic"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/distributor"
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
		distrib:   &distributor.Distributor[*core.Event]{},
		syncPoint: &atomic.Int32{},
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
	writers int32

	cache   individualCache
	distrib *distributor.Distributor[*core.Event]

	syncPoint *atomic.Int32
	mu        *sync.Mutex
}

func (c *sharedCache) observe(m *metric, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.observe(m, v)
}

func (c *sharedCache) flush(out chan<- *core.Event, flushFn func(m *metric, ch chan<- *core.Event)) {
	c.distrib.AppendOut(out)

	if c.syncPoint.Add(1) == c.writers {
		c.flush2(flushFn)
		c.distrib.Reset()
		c.syncPoint.Store(0)
	}
}

func (c *sharedCache) flush2(flushFn func(m *metric, out chan<- *core.Event)) {
	out, done := make(chan *core.Event), make(chan struct{})

	c.mu.Lock()
	defer c.mu.Unlock()

	go func() {
		c.distrib.Run(out)
		close(done)
	}()

	for _, m := range c.cache {
		flushFn(m, out)
		m.reset()
	}

	close(out)
	<-done
}

func (c *sharedCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.clear()
	ss.delete(c.id)
}
