package glob

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gobwas/glob"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Glob struct {
	alias  string
	pipe   string
	RK     []string            `mapstructure:"routing_key"`
	Fields map[string][]string `mapstructure:"fields"`
	Labels map[string][]string `mapstructure:"labels"`

	in       <-chan *core.Event
	accepted chan<- *core.Event
	rejected chan<- *core.Event
	log      *slog.Logger

	noGlobs bool
	rk      []glob.Glob
	fields  map[string][]glob.Glob
	labels  map[string][]glob.Glob
}

func (f *Glob) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, f); err != nil {
		return err
	}

	f.alias = alias
	f.pipe = pipeline
	f.log = log

	if len(f.Fields) == 0 && len(f.Labels) == 0 && len(f.RK) == 0 {
		f.log.Warn("no globs for routing key, fields and labels found")
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

func (f *Glob) Prepare(
	in <-chan *core.Event,
	rejected chan<- *core.Event,
	accepted chan<- *core.Event,
) {
	f.in = in
	f.rejected = rejected
	f.accepted = accepted
}

func (f *Glob) Close() error {
	return nil
}

func (f *Glob) Alias() string {
	return f.alias
}

func (f *Glob) Run() {
	for e := range f.in {
		now := time.Now()
		if f.match(e) {
			f.accepted <- e
			metrics.ObserveFliterSummary("glob", f.alias, f.pipe, metrics.EventAccepted, time.Since(now))
		} else {
			f.rejected <- e
			metrics.ObserveFliterSummary("glob", f.alias, f.pipe, metrics.EventRejected, time.Since(now))
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
		// if event hasn't label, reject it
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
		// if event hasn't field, reject it
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
