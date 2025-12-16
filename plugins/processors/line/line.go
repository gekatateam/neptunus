package line

import (
	"strconv"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type Line struct {
	*core.BaseProcessor `mapstructure:"-"`
	Label               string `mapstructure:"label"`

	line string
}

func (p *Line) Init() error {
	return nil
}

func (p *Line) Close() error {
	return nil
}

func (p *Line) SetLine(line int) {
	p.line = strconv.Itoa(line)
}

func (p *Line) Run() {
	var now time.Time
	for e := range p.In {
		now = time.Now()
		e.SetLabel(p.Label, p.line)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("line", func() core.Processor {
		return &Line{
			Label: "::line",
		}
	})
}
