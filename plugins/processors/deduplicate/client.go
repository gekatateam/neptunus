package deduplicate

import (
	"sync"

	"github.com/redis/go-redis/v9"
)

var clientStorage = &redisClientStorage{
	mu:      &sync.Mutex{},
	clients: make(map[uint64]redis.UniversalClient),
}

type redisClientStorage struct {
	mu      *sync.Mutex
	clients map[uint64]redis.UniversalClient
}

func (s *redisClientStorage) CompareAndStore(id uint64, client redis.UniversalClient) redis.UniversalClient {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.clients[id]; ok {
		client.Close()
		return c
	}

	s.clients[id] = client
	return client
}
