package allowall

import (
	"time"

	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/metrics"
	"github.com/gekatateam/pipeline/plugins"
)

type AllowAll struct {
	alias string
	in       <-chan *core.Event
	accepted chan<- *core.Event
	log      logger.Logger
}

func New(_ map[string]any, alias string, log logger.Logger) (core.Filter, error) {
	return &AllowAll{log: log, alias: alias}, nil
}

func (f *AllowAll) Init(
	in <-chan *core.Event,
	_ chan<- *core.Event,
	accepted chan<- *core.Event,
) {
	f.in = in
	f.accepted = accepted
}

func (f *AllowAll) Close() error {
	return nil
}

func (f *AllowAll) Filter() {
	for e := range f.in {
		now := time.Now()
		f.accepted <- e
		metrics.ObserveFliterSummary("allowall", f.alias, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddFilter("allowall", New)
}
