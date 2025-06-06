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

var targetObjectPattern = regexp.MustCompile(`^((label|field):)?([\w-\.]+)$`)

type Converter struct {
	*core.BaseProcessor `mapstructure:"-"`
	IgnoreOutOfRange    bool     `mapstructure:"ignore_out_of_range"`
	Id                  string   `mapstructure:"id"`
	Label               []string `mapstructure:"label"`
	String              []string `mapstructure:"string"`
	Integer             []string `mapstructure:"integer"`
	Unsigned            []string `mapstructure:"unsigned"`
	Float               []string `mapstructure:"float"`
	Boolean             []string `mapstructure:"boolean"`

	conversions []conversionParams
	converter   *converter
}

func (p *Converter) Init() error {
	if len(p.Id) > 0 {
		if err := p.initConversionParam(p.Id, toId); err != nil {
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
	if len(match) != 4 {
		return fmt.Errorf("configured value %v does not match pattern", rawParam)
	}

	switch match[2] {
	case "label":
		p.conversions = append(p.conversions, conversionParams{
			from: fromLabel,
			to:   to,
			path: match[3],
			ioor: p.IgnoreOutOfRange,
		})
	case "", "field":
		p.conversions = append(p.conversions, conversionParams{
			from: fromField,
			to:   to,
			path: match[3],
			ioor: p.IgnoreOutOfRange,
		})
	default:
		panic(fmt.Errorf("processors.converter: totally unexpected source type: %v", match[2]))
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
