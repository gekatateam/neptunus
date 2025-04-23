package pool

import (
	"sync"
	"time"
)

type Runner[T any] interface {
	Run()
	Push(t T)
	Close() error
}

type Pool[T any, K comparable] struct {
	runners   map[K]Runner[T]
	lastWrite map[K]time.Time
	new       func(key K) Runner[T]
	wg        *sync.WaitGroup
}

func New[T any, K comparable](new func(key K) Runner[T]) *Pool[T, K] {
	return &Pool[T, K]{
		runners:   make(map[K]Runner[T]),
		lastWrite: make(map[K]time.Time),
		wg:        &sync.WaitGroup{},
		new:       new,
	}
}

func (p *Pool[T, K]) Get(key K) Runner[T] {
	if runner, ok := p.runners[key]; ok {
		p.lastWrite[key] = time.Now()
		return runner
	}

	runner := p.new(key)
	p.runners[key] = runner
	p.lastWrite[key] = time.Now()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		runner.Run()
	}()

	return runner
}

func (p *Pool[T, K]) Keys() []K {
	var keys []K
	for k := range p.runners {
		keys = append(keys, k)
	}
	return keys
}

func (p *Pool[T, K]) LastWrite(key K) time.Time {
	return p.lastWrite[key]
}

func (p *Pool[T, K]) Remove(key K) {
	if runner, ok := p.runners[key]; ok {
		runner.Close()
		delete(p.runners, key)
	}
}

func (p *Pool[T, K]) Close() error {
	for key, runner := range p.runners {
		runner.Close()
		delete(p.runners, key)
	}

	p.wg.Wait()
	return nil
}
