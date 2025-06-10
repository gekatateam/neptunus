package syncs

import "sync"

type Map[K comparable, V any] struct {
	m  map[K]V
	mu *sync.Mutex
}

func New[K comparable, V any]() Map[K, V] {
	return Map[K, V]{
		m:  make(map[K]V),
		mu: &sync.Mutex{},
	}
}

func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
}

func (m *Map[K, V]) Store(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[key] = value
}

func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, ok = m.m[key]
	return value, ok
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.m[key]
	if !ok {
		m.m[key] = value
		return value, false
	}
	return v, true
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, v := range m.m {
		if !f(k, v) {
			break
		}
	}
}

func (m *Map[K, V]) Update(key K, f func(value V, loaded bool) V) (actual V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.m[key]
	m.m[key] = f(v, ok)
	return m.m[key]
}
