package sql

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

var clientStorage = &dbClientStorage{
	mu:      &sync.Mutex{},
	clients: make(map[uint64]*sqlx.DB),
	members: make(map[uint64]int),
}

type dbClientStorage struct {
	mu      *sync.Mutex
	clients map[uint64]*sqlx.DB
	members map[uint64]int
}

func (s *dbClientStorage) CompareAndStore(id uint64, client *sqlx.DB) *sqlx.DB {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.members[id]; !ok {
		s.members[id] = 0
	}
	s.members[id]++

	if c, ok := s.clients[id]; ok {
		client.Close()
		return c
	}

	s.clients[id] = client
	return client
}

func (s *dbClientStorage) Leave(id uint64) (last bool) {
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
