package batcher

import (
	"time"

	"github.com/gekatateam/neptunus/core"
)

// batcher is a helper for cases when plugin needs to send events in batches
// it passes a batch of events to passed func when:
// - buffer is full
// - ticker ticks and buffer is not empty
// - input chanel is closed and readed to the end
//
// batcher clears buffer after each call of flushFn()
type Batcher struct {
	Buffer   int
	Interval time.Duration
}

func (b *Batcher) Run(in <-chan *core.Event, flushFn func(buf []*core.Event)) {
	buf := make([]*core.Event, 0, b.Buffer)
	ticker := time.NewTicker(b.Interval)

	for {
		select {
		case e, ok := <-in:
			if !ok { // channel closed
				ticker.Stop()
				if len(buf) > 0 {
					flushFn(buf)
				}
				return
			}

			buf = append(buf, e)
			if len(buf) == b.Buffer { // buffer is full
				flushFn(buf)
				buf = nil
				ticker.Reset(b.Interval)
			}
		case <-ticker.C:
			if len(buf) > 0 {
				flushFn(buf)
				buf = nil
			}
		}
	}
}
