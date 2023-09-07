package line

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Line struct {
	alias string
	pipe  string
	line  string
	Label string `mapstructure:"label"`

	in  <-chan *core.Event
	out chan<- *core.Event
	log *slog.Logger
}

func (p *Line) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log
	p.line = strconv.Itoa(config["::line"].(int))

	return nil
}

func (p *Line) Prepare(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Line) Close() error {
	return nil
}

func (p *Line) Alias() string {
	return p.alias
}

func (p *Line) Run() {
	for e := range p.in {
		now := time.Now()
		e.AddLabel(p.Label, p.line)
		p.out <- e
		metrics.ObserveProcessorSummary("line", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("line", func() core.Processor {
		return &Line{
			Label: "::line",
		}
	})
}
