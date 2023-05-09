package allowall

import (
	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/plugins"
)

type AllowAll struct {
	log logger.Logger
}

func New(config map[string]any, log logger.Logger) (core.Filter, error) {
	return &AllowAll{
		log: log,
	}, nil
}

func (f *AllowAll) Init() error                           { return nil }
func (f *AllowAll) Filter(e ...*core.Event) []*core.Event { return e }
func (f *AllowAll) Close() error                          { return nil }

func init() {
	plugins.AddFilter("allowall", New)
}
