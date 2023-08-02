package batcher

import (
	"time"

	"github.com/gekatateam/neptunus/core"
)

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
				flushFn(buf)
				return
			}

			buf = append(buf, e)
			if len(buf) == b.Buffer { // buffer fullness
				flushFn(buf)
			}
			buf = nil
			ticker.Reset(b.Interval)
		case <-ticker.C:
			flushFn(buf)
			buf = nil
		}
	}
}
