package core

import (
	"errors"
	"strings"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrNotAMap     = errors.New("key is not a map")
)

type Map map[string]any

// key: some.path.to.data
func (m Map) GetValue(key string) (any, error) {
	return findInMap(m, key)
}

func (m Map) SetValue(key string, value any) error {
	return putInMap(m, key, value)
}

func (m Map) DeleteValue(key string) (any, error) {
	return delFromMap(m, key)
}

func (m Map) Clone() Map {
	result := make(Map, len(m))
	for k := range m {
		if nestedMap, ok := anyToMap(m[k]); ok {
			result[k] = nestedMap.Clone()
			continue
		}
		// switch reflect.TypeOf(m[k]).Kind() {
		// case reflect.Array, reflect.Slice:
		// 	reflect.Copy(result[k], m[k])
		// }
		result[k] = m[k]
	}
	return result
}

func anyToMap(v any) (Map, bool) {
	switch m := v.(type) {
	case Map:
		return m, true
	case map[string]any:
		return Map(m), true
	}
	return nil, false
}

func findInMap(m Map, key string) (any, error) {
	for {
		// try to find on the current level;
		// it works with one-level keys like "foo"
		if val, exists := m[key]; exists {
			return val, nil
		}

		// check if key has nested levels
		idx := strings.IndexRune(key, '.')
		if idx < 0 {
			return nil, ErrKeyNotFound
		}

		// get current level from key; "foo.bar" -> "foo"
		k := key[:idx]

		// get next node
		next, exists := m[k]
		if !exists {
			return nil, ErrKeyNotFound
		}

		// try to get next node as Map
		nextMap, ok := anyToMap(next)
		if !ok {
			return nil, ErrKeyNotFound
		}

		// shift key; "foo.bar" -> "bar"
		key = key[idx+1:]
		// shift node for a next iteration
		m = nextMap
	}
}

func putInMap(m Map, key string, value any) error {
	for {
		// check if key has nested levels
		idx := strings.IndexRune(key, '.')
		// if not, set value and return
		if idx < 0 {
			m[key] = value
			return nil
		}

		// get current level from key; "foo.bar" -> "foo"
		k := key[:idx]

		// create next node if it not exists
		if _, exists := m[k]; !exists {
			m[k] = Map(make(map[string]any))
		}

		// try to get next node;
		// if it can't be casted to Map, return without changes
		next, ok := anyToMap(m[k])
		if !ok {
			return ErrNotAMap
		}

		// shift key; "foo.bar" -> "bar"
		key = key[idx+1:]
		// shift node for next iteration
		m = next
	}
}

func delFromMap(m Map, key string) (any, error) {
	for {
		// try to find on the current level;
		// it works with one-level keys like "foo"
		val, exists := m[key]
		if exists {
			delete(m, key)
			return val, nil
		}

		// check if key has nested levels
		idx := strings.IndexRune(key, '.')
		if idx < 0 {
			return nil, ErrKeyNotFound
		}

		// get current level from key; "foo.bar" -> "foo"
		k := key[:idx]

		// get next node
		next, exists := m[k]
		if !exists {
			return nil, ErrKeyNotFound
		}

		// try to get next node as Map
		nextMap, ok := anyToMap(next)
		if !ok {
			return nil, ErrKeyNotFound
		}

		// shift key; "foo.bar" -> "bar"
		key = key[idx+1:]
		// shift node for next iteration
		m = nextMap
	}
}
