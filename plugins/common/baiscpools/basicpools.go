package baiscpools

import (
	"bytes"
	"sync"
)

var BytesBuffer = Pool[*bytes.Buffer]{
	p: &sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, 4096))
		},
	},
}

type Pool[T any] struct{ p *sync.Pool }

func (p Pool[T]) Get() T  { return p.p.Get().(T) }
func (p Pool[T]) Put(x T) { p.p.Put(x) }
