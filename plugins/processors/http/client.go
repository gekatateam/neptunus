package http

import (
	"net/http"
	"sync"
)

var clientStorage = &httpClientStorage{
	mu:      &sync.Mutex{},
	clients: make(map[uint64]*http.Client),
	members: make(map[uint64]int),
}

type httpClientStorage struct {
	mu      *sync.Mutex
	clients map[uint64]*http.Client
	members map[uint64]int
}

func (s *httpClientStorage) CompareAndStore(id uint64, client *http.Client) *http.Client {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.members[id]; !ok {
		s.members[id] = 0
	}
	s.members[id]++

	if c, ok := s.clients[id]; ok {
		return c
	}

	s.clients[id] = client
	return client
}

func (s *httpClientStorage) Leave(id uint64) (last bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.members[id]--
	if s.members[id] == 0 {
		delete(s.clients, id)
		delete(s.members, id)
		return true
	}

	return false
}
