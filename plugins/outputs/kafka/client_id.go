package kafka

import "sync"

var checker = &idChecker{
	s:  make(map[string]struct{}),
	mu: &sync.Mutex{},
}

type idChecker struct {
	s  map[string]struct{}
	mu *sync.Mutex
}

func (i *idChecker) set(clientId string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, ok := i.s[clientId]; ok {
		return false
	}

	i.s[clientId] = struct{}{}
	return true
}
