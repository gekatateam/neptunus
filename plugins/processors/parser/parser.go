package parser

import (
	"errors"
	"fmt"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/ider"
)

type Parser struct {
	*core.BaseProcessor `mapstructure:"-"`
	Behaviour           string `mapstructure:"behaviour"`
	From                string `mapstructure:"from"`
	To                  string `mapstructure:"to"`
	DropOrigin          bool   `mapstructure:"drop_origin"`
	*ider.Ider          `mapstructure:",squash"`

	parser core.Parser
}

func (p *Parser) Init() error {
	if len(p.From) == 0 {
		return errors.New("from field required")
	}

	if len(p.To) == 0 {
		p.To = "."
	}

	if err := p.Ider.Init(); err != nil {
		return err
	}

	switch p.Behaviour {
	case "merge", "produce":
	default:
		return fmt.Errorf("forbidden behaviour: %v; expected one of: merge, produce", p.Behaviour)
	}

	return nil
}

func (p *Parser) Close() error {
	return nil
}

func (p *Parser) SetParser(parser core.Parser) {
	p.parser = parser
}

func (p *Parser) Run() {
MAIN_LOOP:
	for e := range p.In {
		now := time.Now()

		rawField, err := e.GetField(p.From)
		if err != nil {
			p.Out <- e
			p.Observe(metrics.EventAccepted, time.Since(now))
			continue // do nothing if event has no field
		}

		var field []byte
		switch t := rawField.(type) {
		case string:
			field = []byte(t)
		case []byte:
			field = t
		default:
			p.Out <- e
			p.Observe(metrics.EventAccepted, time.Since(now))
			continue // do nothing if field is not a string or bytes slice
		}

		events, err := p.parser.Parse(field, e.RoutingKey)
		if err != nil {
			p.Log.Error("parsing failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue // continue with error if parsing failed
		}

		switch p.Behaviour {
		case "produce":
			for _, donor := range events {
				// here we using origin event copy instead of
				// one of returned by parser because it's easyest way
				// to copy origin labels and tags
				// and make sure than payload
				// will be tracked as well
				event := e.Clone()
				event.Data = donor.Data
				p.Ider.Apply(event)
				p.Out <- event
			}
			if p.DropOrigin {
				p.Drop <- e
			} else {
				p.Out <- e
			}
			p.Log.Debug(fmt.Sprintf("produced %v events", len(events)))
		case "merge":
			for _, donor := range events {
				// and again, we using event copy, same reason
				event := e.Clone()
				if p.DropOrigin {
					event.DeleteField(p.From)
				}

				if err := event.SetField(p.To, donor.Data); err != nil {
					p.Drop <- event // drop cloned event
					p.Log.Error("error set field",
						"error", fmt.Errorf("%v: %w", p.To, err),
						elog.EventGroup(e),
					)
					e.StackError(fmt.Errorf("error set field: %w", err))
					p.Out <- e
					p.Observe(metrics.EventFailed, time.Since(now))
					continue MAIN_LOOP // continue main loop with error if set failed
				}
				p.Out <- event
			}
			p.Drop <- e // because we no longer need the origin event
			p.Log.Debug(fmt.Sprintf("produced %v events", len(events)))
		}

		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("parser", func() core.Processor {
		return &Parser{
			Behaviour:  "merge",
			DropOrigin: true,
			To:         ".",
			Ider:       &ider.Ider{},
		}
	})
}
