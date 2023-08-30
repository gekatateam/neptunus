package distributor

type Distributor[T any] struct {
	outs []chan <- T
}

func (d *Distributor[T]) AppendOut(ch chan <- T) {
	d.outs = append(d.outs, ch)
}

func (d *Distributor[T]) Run(ch <- chan T) {
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
