package sql

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

var clientStorage = &dbClientStorage{
	mu:      &sync.Mutex{},
	clients: make(map[uint64]*sqlx.DB),
}

type dbClientStorage struct {
	mu      *sync.Mutex
	clients map[uint64]*sqlx.DB
}

func (s *dbClientStorage) CompareAndStore(id uint64, client *sqlx.DB) *sqlx.DB {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.clients[id]; ok {
		client.Close()
		return c
	}

	s.clients[id] = client
	return client
}
