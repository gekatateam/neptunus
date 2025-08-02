package sharedstorage

import (
	"io"
	"sync"
)

type SharedStorage[T any, K comparable] struct {
	mu      *sync.Mutex
	objects map[K]T
	members map[K]int
}

func New[T any, K comparable]() *SharedStorage[T, K] {
	return &SharedStorage[T, K]{
		mu:      &sync.Mutex{},
		objects: make(map[K]T),
		members: make(map[K]int),
	}
}

func (s *SharedStorage[T, K]) CompareAndStore(id K, obj T) T {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.members[id]; !ok {
		s.members[id] = 0
	}
	s.members[id]++

	if c, ok := s.objects[id]; ok {
		if closer, ok := any(obj).(io.Closer); ok {
			closer.Close()
		}
		return c
	}

	s.objects[id] = obj
	return obj
}

func (s *SharedStorage[T, K]) Leave(id K) (last bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.members[id]--
	if s.members[id] == 0 {
		delete(s.objects, id)
		delete(s.members, id)
		return true
	}

	return false
}
