package batcher

import (
	"time"
)

// batcher is a helper for cases when plugin needs to send events in batches
// it passes a batch of events to passed func when:
// - buffer is full
// - ticker ticks
// - input chanel is closed and readed to the end
//
// batcher clears buffer after each call of flushFn()
type Batcher[T any] struct {
	Buffer   int           `mapstructure:"batch_buffer"`
	Interval time.Duration `mapstructure:"batch_interval"`
}

func (b *Batcher[T]) Run(in <-chan T, flushFn func(buf []T)) {
	buf := make([]T, 0, b.Buffer)
	ticker := time.NewTicker(b.Interval)

	for {
		select {
		case e, ok := <-in:
			if !ok { // channel closed
				ticker.Stop()
				flushFn(buf)
				return
			}

			buf = append(buf, e)
			if len(buf) == b.Buffer { // buffer is full
				flushFn(buf)
				buf = nil
				ticker.Reset(b.Interval)
			}
		case <-ticker.C:
			flushFn(buf)
			buf = nil
		}
	}
}
