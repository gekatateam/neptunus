package allowall

import (
	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/plugins"
)

type AllowAll struct {
	in       <-chan *core.Event
	accepted chan<- *core.Event
	log      logger.Logger
}

func New(_ map[string]any, log logger.Logger) (core.Filter, error) {
	return &AllowAll{
		log: log,
	}, nil
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
		f.accepted <- e
	}
}

func init() {
	plugins.AddFilter("allowall", New)
}
