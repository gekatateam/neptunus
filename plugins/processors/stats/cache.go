package stats

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"

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
		cache.sem = semaphore.NewWeighted(cache.writers)
		return cache
	}

	cache := &sharedCache{
		id:        k,
		writers:   1,
		cache:     newIndividualCache(),
		distr:     distributor.New[*core.Event](),
		syncPoint: 0,
		sem:       semaphore.NewWeighted(1),
		wg:        &sync.WaitGroup{},
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
	distr *distributor.Distributor[*core.Event]

	syncPoint int64
	sem       *semaphore.Weighted
	wg        *sync.WaitGroup
}

func (c *sharedCache) observe(m *metric, v float64) {
	c.sem.Acquire(context.Background(), c.writers)
	defer c.sem.Release(c.writers)

	c.cache.observe(m, v)
}

func (c *sharedCache) flush(out chan<- *core.Event, flushFn func(m *metric, ch chan<- *core.Event)) {
	c.sem.Acquire(context.Background(), 1)
	defer c.sem.Release(1)

	c.wg.Add(1)
	c.distr.AppendOut(out)

	c.syncPoint += 1
	if c.syncPoint == c.writers {
		c.flush2(flushFn)
		c.distr.Reset()
		c.syncPoint = 0
		c.wg.Add(-int(c.writers))
	}

	c.wg.Wait()
}

func (c *sharedCache) flush2(flushFn func(m *metric, out chan<- *core.Event)) {
	out, done := make(chan *core.Event), make(chan struct{})

	go func() {
		c.distr.Run(out)
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
	c.sem.Acquire(context.Background(), c.writers)
	defer c.sem.Release(c.writers)

	if c.syncPoint == 0 {
		c.cache.clear()
		ss.delete(c.id)
	}
}
