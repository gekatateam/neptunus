package sharedstorage

import (
	"io"
	"sync"
)

type SharedStorage[T any] struct {
	mu      *sync.Mutex
	objects map[uint64]T
	members map[uint64]int
}

func New[T any]() *SharedStorage[T] {
	return &SharedStorage[T]{
		mu:      &sync.Mutex{},
		objects: make(map[uint64]T),
		members: make(map[uint64]int),
	}
}

func (s *SharedStorage[T]) CompareAndStore(id uint64, obj T) T {
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

func (s *SharedStorage[T]) Leave(id uint64) (last bool) {
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
