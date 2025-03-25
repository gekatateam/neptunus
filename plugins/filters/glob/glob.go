package glob

import (
	"fmt"
	"time"

	"github.com/gobwas/glob"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Glob struct {
	*core.BaseFilter `mapstructure:"-"`
	RK               []string            `mapstructure:"routing_key"`
	Fields           map[string][]string `mapstructure:"fields"`
	Labels           map[string][]string `mapstructure:"labels"`

	noGlobs bool
	rk      []glob.Glob
	fields  map[string][]glob.Glob
	labels  map[string][]glob.Glob
}

func (f *Glob) Init() error {
	if len(f.Fields) == 0 && len(f.Labels) == 0 && len(f.RK) == 0 {
		f.Log.Warn("no globs for routing key, fields and labels found")
		f.noGlobs = true
	}
	f.fields = make(map[string][]glob.Glob, len(f.Fields))
	f.labels = make(map[string][]glob.Glob, len(f.Labels))

	for _, value := range f.RK {
		glob, err := glob.Compile(value)
		if err != nil {
			return fmt.Errorf("routing key glob %v compilation failed: %v", value, err.Error())
		}
		f.rk = append(f.rk, glob)
	}

	for key, values := range f.Fields {
		for _, value := range values {
			glob, err := glob.Compile(value)
			if err != nil {
				return fmt.Errorf("field glob %v:%v compilation failed: %v", key, value, err.Error())
			}
			f.fields[key] = append(f.fields[key], glob)
		}
	}

	for key, values := range f.Labels {
		for _, value := range values {
			glob, err := glob.Compile(value)
			if err != nil {
				return fmt.Errorf("label glob %v:%v compilation failed: %v", key, value, err.Error())
			}
			f.labels[key] = append(f.labels[key], glob)
		}
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
	if len(f.rk) > 0 {
		if !f.matchAny(f.rk, e.RoutingKey) {
			return false
		}
	}

	// check labels
	for key, globs := range f.labels {
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
	for key, globs := range f.fields {
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
