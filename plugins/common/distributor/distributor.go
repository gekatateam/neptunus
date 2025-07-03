package distributor

import "sync"

// distributor is a helper for cases when plugin
// needs to distribute events between multiple channels
type Distributor[T any] struct {
	outs []chan<- T
	mu   *sync.Mutex
}

func New[T any]() *Distributor[T] {
	return &Distributor[T]{
		outs: make([]chan<- T, 0),
		mu:   &sync.Mutex{},
	}
}

func (d *Distributor[T]) AppendOut(ch chan<- T) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.outs = append(d.outs, ch)
}

func (d *Distributor[T]) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.outs = make([]chan<- T, 0, len(d.outs))
}

func (d *Distributor[T]) Run(ch <-chan T) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.outs) == 0 {
		panic("distributor started with no outputs")
	}

	cursor := 0
	for e := range ch {
		d.outs[cursor] <- e

		cursor += 1
		if cursor > len(d.outs)-1 {
			cursor = 0
		}
	}
}
