package glob

import (
	"fmt"
	"time"

	"github.com/gobwas/glob"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
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
	log      logger.Logger

	rk     []glob.Glob
	fields map[string][]glob.Glob
	labels map[string][]glob.Glob
}

func New(config map[string]any, alias, pipeline string, log logger.Logger) (core.Filter, error) {
	g := &Glob{
		log:   log,
		alias: alias,
		pipe:  pipeline,
	}

	if err := mapstructure.Decode(config, g); err != nil {
		return nil, err
	}

	if len(g.Fields) == 0 && len(g.Labels) == 0 && len(g.RK) == 0 {
		g.log.Warn("no globs for routing key, fields and labels found")
	}
	g.fields = make(map[string][]glob.Glob, len(g.Fields))
	g.labels = make(map[string][]glob.Glob, len(g.Labels))

	for _, value := range g.RK {
		glob, err := glob.Compile(value)
		if err != nil {
			return nil, fmt.Errorf("routing key glob %v compilation failed: %v", value, err.Error())
		}
		g.rk = append(g.rk, glob)
	}

	for key, values := range g.Fields {
		for _, value := range values {
			glob, err := glob.Compile(value)
			if err != nil {
				return nil, fmt.Errorf("field glob %v:%v compilation failed: %v", key, value, err.Error())
			}
			g.fields[key] = append(g.fields[key], glob)
		}
	}

	for key, values := range g.Labels {
		for _, value := range values {
			glob, err := glob.Compile(value)
			if err != nil {
				return nil, fmt.Errorf("label glob %v:%v compilation failed: %v", key, value, err.Error())
			}
			g.labels[key] = append(g.labels[key], glob)
		}
	}

	return g, nil
}

func (f *Glob) Init(
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

func (f *Glob) Filter() {
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
	if len(f.fields) == 0 && len(f.labels) == 0 && len(f.rk) == 0 {
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
	plugins.AddFilter("glob", New)
}
