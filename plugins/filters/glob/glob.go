package glob

import (
	"time"

	"github.com/gobwas/glob"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Glob struct {
	*core.BaseFilter `mapstructure:"-"`
	RK               []glob.Glob            `mapstructure:"routing_key"`
	Fields           map[string][]glob.Glob `mapstructure:"fields"`
	Labels           map[string][]glob.Glob `mapstructure:"labels"`

	noGlobs bool
}

func (f *Glob) Init() error {
	if len(f.Fields) == 0 && len(f.Labels) == 0 && len(f.RK) == 0 {
		f.Log.Warn("no globs for routing key, fields and labels found")
		f.noGlobs = true
	}

	return nil
}

func (f *Glob) Close() error {
	return nil
}

func (f *Glob) Run() {
	for e := range f.In {
		now := time.Now()
		if f.match(e) {
			f.Acc <- e
			f.Observe(metrics.EventAccepted, time.Since(now))
		} else {
			f.Rej <- e
			f.Observe(metrics.EventRejected, time.Since(now))
		}
	}
}

func (f *Glob) match(e *core.Event) bool {
	// pass event if no filters are set
	if f.noGlobs {
		return true
	}

	// check routing key
	if f.RK != nil {
		if !f.matchAny(f.RK, e.RoutingKey) {
			return false
		}
	}

	// check labels
	for key, globs := range f.Labels {
		// if event doesn't have label, reject it
		label, ok := e.GetLabel(key)
		if !ok {
			return false
		}
		// if label not matched any glob, reject it
		if !f.matchAny(globs, label) {
			return false
		}
	}

	// check fields
	for key, globs := range f.Fields {
		// if event doesn't have field, reject it
		fieldRaw, err := e.GetField(key)
		if err != nil {
			return false
		}
		// if field not a string, reject event
		field, ok := fieldRaw.(string)
		if !ok {
			return false
		}
		// if field not matched any glob, reject it
		if !f.matchAny(globs, field) {
			return false
		}
	}

	return true
}

func (f *Glob) matchAny(globs []glob.Glob, value string) bool {
	for _, glob := range globs {
		if glob.Match(value) {
			return true
		}
	}
	return false
}

func init() {
	plugins.AddFilter("glob", func() core.Filter {
		return &Glob{}
	})
}
