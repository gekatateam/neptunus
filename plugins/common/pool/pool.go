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

type Pool[T any] struct {
	runners   map[string]Runner[T]
	lastWrite map[string]time.Time
	new       func(key string) Runner[T]
	wg        *sync.WaitGroup
}

func New[T any](new func(key string) Runner[T]) *Pool[T] {
	return &Pool[T]{
		runners:   make(map[string]Runner[T]),
		lastWrite: make(map[string]time.Time),
		wg:        &sync.WaitGroup{},
		new:       new,
	}
}

func (p *Pool[T]) Get(key string) Runner[T] {
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

func (p *Pool[T]) Keys() []string {
	var keys []string
	for k := range p.runners {
		keys = append(keys, k)
	}
	return keys
}

func (p *Pool[T]) LastWrite(key string) time.Time {
	return p.lastWrite[key]
}

func (p *Pool[T]) Remove(key string) {
	if runner, ok := p.runners[key]; ok {
		runner.Close()
		delete(p.runners, key)
	}
}

func (p *Pool[T]) Close() error {
	for key, runner := range p.runners {
		runner.Close()
		delete(p.runners, key)
	}

	p.wg.Wait()
	return nil
}
