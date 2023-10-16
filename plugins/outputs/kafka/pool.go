package kafka

import (
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
)

type WritersPool struct {
	chans map[string]*writer
	mu    *sync.RWMutex
}

type writer struct {
	input chan *core.Event
	w     *kafka.Writer
	t     *time.Timer
	*batcher.Batcher[*core.Event]
}

func (w *writer) Run()
