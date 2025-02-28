package http

import (
	"net/http"
	"sync"
)

var clientStorage = &httpClientStorage{
	mu:      &sync.Mutex{},
	clients: make(map[uint64]*http.Client),
}

type httpClientStorage struct {
	mu      *sync.Mutex
	clients map[uint64]*http.Client
}

func (s *httpClientStorage) CompareAndStore(id uint64, client *http.Client) *http.Client {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.clients[id]; ok {
		return c
	}

	s.clients[id] = client
	return client
}
