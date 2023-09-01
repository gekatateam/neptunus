package distributor

// distributor is a helper for cases when plugin 
// needs to distribute events between multiple channels
type Distributor[T any] struct {
	outs []chan<- T
}

func (d *Distributor[T]) AppendOut(ch chan<- T) {
	d.outs = append(d.outs, ch)
}

func (d *Distributor[T]) Reset() {
	d.outs = nil
}

func (d *Distributor[T]) Run(ch <-chan T) {
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
