package core

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrInvalidPath  = errors.New("invalid path")
	ErrNoSuchField  = errors.New("no such field on the provided path") // this error should be returned in any case
)

// map[string]any or []any
type Payload any

func FindInPayload(p Payload, key string) (any, error) {
	if len(key) == 0 {
		return nil, ErrInvalidPath
	}

	if key == "." {
		return p, nil
	}

	if key[0] == '.' {
		return nil, ErrInvalidPath
	}

	for {
		dotIndex := strings.IndexRune(key, '.')
		if dotIndex < 0 { // no nested keys
			node, err := searchInNode(p, key)
			if err != nil {
				return nil, ErrNoSuchField
			}

			return node, nil
		}

		next, err := searchInNode(p, key[:dotIndex])
		if err != nil {
			return nil, ErrNoSuchField
		}

		key = key[dotIndex+1:] // shift path: foo.bar.buzz -> bar.buzz
		p = next // shift searchable object
	}
}

func PutInPayload(p Payload, key string, val any) (Payload, error) {
	if len(key) == 0 {
		return nil, ErrInvalidPath
	}

	if key == "." {
		if p == nil {
			return val, nil
		}

		if pl, ok := p.(map[string]any); ok {
			if vl, ok := val.(map[string]any); ok {
				for k, v := range vl {
					pl[k] = v
				}
				return pl, nil
			}
		}

		if pl, ok := p.([]any); ok {
			if vl, ok := val.([]any); ok {
				return append(pl, vl...), nil
			}
		}

		return nil, ErrInvalidPath
	}

	if key[0] == '.' {
		return nil, ErrInvalidPath
	}

	return putInPayload(p, key, val)
}

func DeleteFromPayload(p Payload, key string) (Payload, error) {
	if len(key) == 0 {
		return nil, ErrInvalidPath
	}

	if key == "." {
		return nil, nil
	}

	if key[0] == '.' {
		return nil, ErrInvalidPath
	}

	return deleteFromPayload(p, key)
}

func ClonePayload(p Payload) Payload {
	switch t := p.(type) {
	case map[string]any:
		m := make(map[string]any)
		for k := range t {
			m[k] = ClonePayload(t[k])
		}
		return m
	case []any:
		s := make([]any, len(t))
		for i := range t {
			s[i] = ClonePayload(t[i])
		}
		return s
	default:
		return p
	}
}

func putInPayload(p Payload, key string, val any) (Payload, error) {
	dotIndex := strings.IndexRune(key, '.')
	if dotIndex < 0 { // no nested keys
		if p == nil {
			p = createNode(key)
		}
		return putInNode(p, key, val)
	}

	currNode := p
	currKey := key[:dotIndex]

	if currNode == nil {
		currNode = createNode(currKey)
	}

	nextKey := key[dotIndex+1:]
	nextNode, err := searchInNode(currNode, currKey)
	if err == ErrInvalidPath {
		return nil, err
	}

	nextNode, err = putInPayload(nextNode, nextKey, val)
	if err != nil {
		return nil, err
	}

	return putInNode(currNode, currKey, nextNode)
}

func deleteFromPayload(p Payload, key string) (Payload, error) {
	dotIndex := strings.IndexRune(key, '.')
	if dotIndex < 0 { // no nested keys
		if p == nil {
			p = createNode(key)
		}
		return deleteFromNode(p, key)
	}

	currNode := p
	currKey := key[:dotIndex]

	nextKey := key[dotIndex+1:]
	nextNode, err := searchInNode(currNode, currKey)
	if err != nil {
		return nil, err
	}

	nextNode, err = deleteFromPayload(nextNode, nextKey)
	if err != nil {
		return nil, err
	}

	return putInNode(currNode, currKey, nextNode)
}

func searchInNode(p Payload, key string) (Payload, error) {
	switch t := p.(type) {
	case map[string]any:
		if val, ok := t[key]; ok {
			return val, nil
		} else {
			return nil, ErrNoSuchField
		}
	case []any:
		i, err := strconv.Atoi(key)
		if err != nil || i < 0 {
			return nil, ErrNoSuchField
		}

		if i < len(t) {
			return t[i], nil
		}

		return nil, ErrNoSuchField
	default:
		return nil, ErrInvalidPath
	}
}

func createNode(key string) (Payload) {
	if i, err := strconv.Atoi(key); err == nil && i >= 0 {
		s := make([]any, i+1, i+1)
		return s
	}

	m := make(map[string]any)
	return m
}

func putInNode(p Payload, key string, val any) (Payload, error) {
	switch t := p.(type) {
	case map[string]any:
		t[key] = val
		return t, nil
	case []any:
		i, err := strconv.Atoi(key)
		if err != nil || i < 0 {
			return nil, ErrInvalidPath
		}

		if i < len(t) {
			t[i] = val
			return t, nil
		}

		n := slices.Grow(t, i+1-len(t))
		for j := len(n); j <= i; j++ {
			n = append(n, nil)
		}
		n[i] = val
		return n, nil
	default:
		return nil, ErrInvalidPath
	}
}

func deleteFromNode(p Payload, key string) (Payload, error) {
	switch t := p.(type) {
	case map[string]any:
		if _, ok := t[key]; ok {
			delete(t, key)
			return t, nil
		}
		return nil, ErrNoSuchField
	case []any:
		i, err := strconv.Atoi(key)
		if err != nil || i < 0 {
			return nil, ErrInvalidPath
		}

		if i < len(t) {
			return append(t[:i], t[i+1:]...), nil
		}

		return nil, ErrNoSuchField
	default:
		return nil, ErrInvalidPath
	}
}
