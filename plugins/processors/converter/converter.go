package converter

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

// https://go.dev/play/p/zvCYfzqA78O
var targetObjectPattern = regexp.MustCompile(`^((label|field|id|uuid|timestamp|routingkey):)?([\w\.-]+):?(.+)?$`)

type Converter struct {
	*core.BaseProcessor `mapstructure:"-"`
	IgnoreOutOfRange    bool     `mapstructure:"ignore_out_of_range"`
	Id                  string   `mapstructure:"id"`
	Timestamp           string   `mapstructure:"timestamp"`
	RoutingKey          string   `mapstructure:"routing_key"`
	Label               []string `mapstructure:"label"`
	String              []string `mapstructure:"string"`
	Integer             []string `mapstructure:"integer"`
	Unsigned            []string `mapstructure:"unsigned"`
	Float               []string `mapstructure:"float"`
	Boolean             []string `mapstructure:"boolean"`
	Time                []string `mapstructure:"time"`
	Duration            []string `mapstructure:"duration"`

	conversions []conversionParams
	converter   *converter
}

func (p *Converter) Init() error {
	if len(p.Id) > 0 {
		if err := p.initConversionParam(p.Id, toId); err != nil {
			return fmt.Errorf("id: %w", err)
		}
	}

	if len(p.Timestamp) > 0 {
		if err := p.initConversionParam(p.Id, toTimestamp); err != nil {
			return fmt.Errorf("id: %w", err)
		}
	}

	if len(p.RoutingKey) > 0 {
		if err := p.initConversionParam(p.Id, toRoutingKey); err != nil {
			return fmt.Errorf("id: %w", err)
		}
	}

	for _, v := range p.Label {
		if err := p.initConversionParam(v, toLabel); err != nil {
			return fmt.Errorf("label: %w", err)
		}
	}

	for _, v := range p.String {
		if err := p.initConversionParam(v, toString); err != nil {
			return fmt.Errorf("string: %w", err)
		}
	}

	for _, v := range p.Integer {
		if err := p.initConversionParam(v, toInteger); err != nil {
			return fmt.Errorf("integer: %w", err)
		}
	}

	for _, v := range p.Unsigned {
		if err := p.initConversionParam(v, toUnsigned); err != nil {
			return fmt.Errorf("unsigned: %w", err)
		}
	}

	for _, v := range p.Float {
		if err := p.initConversionParam(v, toFloat); err != nil {
			return fmt.Errorf("float: %w", err)
		}
	}

	for _, v := range p.Boolean {
		if err := p.initConversionParam(v, toBoolean); err != nil {
			return fmt.Errorf("boolean: %w", err)
		}
	}

	p.converter = &converter{}

	return nil
}

func (p *Converter) initConversionParam(rawParam string, to to) error {
	match := targetObjectPattern.FindStringSubmatch(rawParam)
	if len(match) != 6 {
		return fmt.Errorf("configured value %v does not match pattern", rawParam)
	}

	switch match[2] {
	case "id":
		p.conversions = append(p.conversions, conversionParams{
			from: fromId,
			to:   to,
			path: match[3],
			tlyt: defaultLayout(match[5]),
			ioor: p.IgnoreOutOfRange,
		})
	case "uuid":
		p.conversions = append(p.conversions, conversionParams{
			from: fromUuid,
			to:   to,
			path: match[3],
			tlyt: defaultLayout(match[5]),
			ioor: p.IgnoreOutOfRange,
		})
	case "timestamp":
		p.conversions = append(p.conversions, conversionParams{
			from: fromTimestamp,
			to:   to,
			path: match[3],
			tlyt: defaultLayout(match[5]),
			ioor: p.IgnoreOutOfRange,
		})
	case "routingkey":
		p.conversions = append(p.conversions, conversionParams{
			from: fromRoutingKey,
			to:   to,
			path: match[3],
			tlyt: defaultLayout(match[5]),
			ioor: p.IgnoreOutOfRange,
		})
	case "label":
		p.conversions = append(p.conversions, conversionParams{
			from: fromLabel,
			to:   to,
			path: match[3],
			tlyt: defaultLayout(match[5]),
			ioor: p.IgnoreOutOfRange,
		})
	case "", "field":
		p.conversions = append(p.conversions, conversionParams{
			from: fromField,
			to:   to,
			path: match[3],
			tlyt: defaultLayout(match[5]),
			ioor: p.IgnoreOutOfRange,
		})
	default:
		return fmt.Errorf("processors.converter: unexpected source type: %v", match[2])
	}

	return nil
}

func (p *Converter) Close() error {
	return nil
}

func (p *Converter) Run() {
	for e := range p.In {
		now := time.Now()
		var hasError bool

		for _, c := range p.conversions {
			if err := p.converter.Convert(e, c); err != nil {
				p.Log.Error("conversion failed",
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(err)
				hasError = true
			}
		}

		p.Out <- e
		if hasError {
			p.Observe(metrics.EventFailed, time.Since(now))
		} else {
			p.Observe(metrics.EventAccepted, time.Since(now))
		}
	}
}

func init() {
	plugins.AddProcessor("converter", func() core.Processor {
		return &Converter{}
	})
}
