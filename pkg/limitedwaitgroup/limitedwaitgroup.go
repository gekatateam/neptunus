package limitedwaitgroup

import "sync"

type LimitedWaitGroup struct {
	ch chan struct{}
	wg *sync.WaitGroup
}

func New(capacity int) *LimitedWaitGroup {
	if capacity <= 0 {
		capacity = 1
	}

	return &LimitedWaitGroup{
		ch: make(chan struct{}, capacity),
		wg: &sync.WaitGroup{},
	}
}

func (s *LimitedWaitGroup) Add() {
	s.ch <- struct{}{}
	s.wg.Add(1)
}

func (s *LimitedWaitGroup) Done() {
	<-s.ch
	s.wg.Done()
}

func (s *LimitedWaitGroup) Len() int {
	return len(s.ch)
}

func (s *LimitedWaitGroup) Cap() int {
	return cap(s.ch)
}

func (s *LimitedWaitGroup) Wait() {
	s.wg.Wait()
}
