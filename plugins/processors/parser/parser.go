package parser

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Parser struct {
	alias      string
	pipe       string
	Behaviour  string `mapstructure:"behaviour"`
	From       string `mapstructure:"from"`
	To         string `mapstructure:"to"`
	DropOrigin bool   `mapstructure:"drop_origin"`

	parser core.Parser
	in     <-chan *core.Event
	out    chan<- *core.Event
	log    *slog.Logger
}

func (p *Parser) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, p); err != nil {
		return err
	}

	if len(p.From) == 0 {
		return errors.New("target field required")
	}

	switch p.Behaviour {
	case "merge", "produce":
	default:
		return fmt.Errorf("forbidden behaviour: %v; expected one of: merge, produce", p.Behaviour)
	}

	p.alias = alias
	p.pipe = pipeline
	p.log = log

	return nil
}

func (p *Parser) Prepare(
	in <-chan *core.Event,
	out chan<- *core.Event,
) {
	p.in = in
	p.out = out
}

func (p *Parser) Close() error {
	return nil
}

func (p *Parser) Alias() string {
	return p.alias
}

func (p *Parser) SetParser(parser core.Parser) {
	p.parser = parser
}

func (p *Parser) Run() {
MAIN_LOOP:
	for e := range p.in {
		now := time.Now()

		rawField, err := e.GetField(p.From)
		if err != nil {
			p.out <- e
			metrics.ObserveProcessorSummary("parser", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
			continue // do nothing if event has no field
		}

		var field []byte
		switch t := rawField.(type) {
		case string:
			field = []byte(t)
		case []byte:
			field = t
		default:
			p.out <- e
			metrics.ObserveProcessorSummary("parser", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
			continue // do nothing if field is not a string or bytes slice
		}

		events, err := p.parser.Parse(field, e.RoutingKey)
		if err != nil {
			p.log.Error("parsing failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(err)
			e.AddTag("::parser_processing_failed")
			p.out <- e
			metrics.ObserveProcessorSummary("parser", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
			continue // continue with error if parsing failed
		}

		switch p.Behaviour {
		case "produce":
			for _, donor := range events {
				// here we using origin event copy instead of
				// one of returned by parser because it's easyest way
				// to copy origin labels and tags
				event := e.Copy()
				event.Data = donor.Data
				p.out <- event
			}
			if !p.DropOrigin {
				p.out <- e
			}
			p.log.Debug(fmt.Sprintf("produced %v events", len(events)))
		case "merge":
			for _, donor := range events {
				event := e.Copy()
				if p.DropOrigin {
					event.DeleteField(p.From)
				}

				if p.To == "" {
					event.AppendFields(donor.Data)
				} else {
					if err := event.SetField(p.To, donor.Data); err != nil {
						p.log.Error("error set field",
							"error", err,
							slog.Group("event",
								"id", e.Id,
								"key", e.RoutingKey,
								"field", p.To,
							),
						)
						e.StackError(fmt.Errorf("error set to field %v: %v", p.To, err))
						e.AddTag("::parser_processing_failed")
						p.out <- e
						metrics.ObserveProcessorSummary("parser", p.alias, p.pipe, metrics.EventFailed, time.Since(now))
						continue MAIN_LOOP // continue main loop with error if set failed
					}
				}
				p.out <- event
			}
			p.log.Debug(fmt.Sprintf("produced %v events", len(events)))
		}

		metrics.ObserveProcessorSummary("parser", p.alias, p.pipe, metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("parser", func() core.Processor {
		return &Parser{
			Behaviour:  "merge",
			DropOrigin: true,
			To:         "",
		}
	})
}
