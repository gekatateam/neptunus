package pass

import (
	"time"

	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/metrics"
	"github.com/gekatateam/pipeline/plugins"
)

type Pass struct {
	alias    string
	in       <-chan *core.Event
	accepted chan<- *core.Event
	log      logger.Logger
}

func New(_ map[string]any, alias string, log logger.Logger) (core.Filter, error) {
	return &Pass{log: log, alias: alias}, nil
}

func (f *Pass) Init(
	in <-chan *core.Event,
	_ chan<- *core.Event,
	accepted chan<- *core.Event,
) {
	f.in = in
	f.accepted = accepted
}

func (f *Pass) Close() error {
	return nil
}

func (f *Pass) Filter() {
	for e := range f.in {
		now := time.Now()
		f.accepted <- e
		metrics.ObserveFliterSummary("pass", f.alias, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddFilter("pass", New)
}
