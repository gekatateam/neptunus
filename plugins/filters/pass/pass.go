package pass

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Pass struct {
	alias    string
	pipe     string
	in       <-chan *core.Event
	accepted chan<- *core.Event
	log      logger.Logger
}

func New(_ map[string]any, alias, pipeline string, log logger.Logger) (core.Filter, error) {
	return &Pass{
		log:   log,
		alias: alias,
		pipe:  pipeline,
	}, nil
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

func (f *Pass) Alias() string {
	return f.alias
}

func (f *Pass) Filter() {
	for e := range f.in {
		now := time.Now()
		f.accepted <- e
		metrics.ObserveFliterSummary("pass", f.alias, f.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddFilter("pass", New)
}
