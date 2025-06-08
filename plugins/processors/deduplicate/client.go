package deduplicate

import (
	"sync"

	"github.com/redis/go-redis/v9"
)

var clientStorage = &redisClientStorage{
	mu:      &sync.Mutex{},
	clients: make(map[uint64]redis.UniversalClient),
	members: make(map[uint64]int),
}

type redisClientStorage struct {
	mu      *sync.Mutex
	clients map[uint64]redis.UniversalClient
	members map[uint64]int
}

func (s *redisClientStorage) CompareAndStore(id uint64, client redis.UniversalClient) redis.UniversalClient {
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

func (s *redisClientStorage) Leave(id uint64) (last bool) {
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
