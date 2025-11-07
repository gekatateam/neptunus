package switchcase

import (
	"fmt"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
)

type Mapping map[string][]string

type SwitchCase struct {
	*core.BaseProcessor `mapstructure:"-"`
	Labels              map[string]Mapping `mapstructure:"labels"`
	Fields              map[string]Mapping `mapstructure:"fields"`
	labelsIndex         map[string]map[string]string
	fieldsIndex         map[string]map[string]string
}

func (p *SwitchCase) Init() error {
	p.labelsIndex = make(map[string]map[string]string, len(p.Labels))
	for label, mapping := range p.Labels {
		index, err := p.mappingToIndex(mapping)
		if err != nil {
			return fmt.Errorf("label %v: %w", label, err)
		}

		p.labelsIndex[label] = index
	}

	p.fieldsIndex = make(map[string]map[string]string, len(p.Fields))
	for field, mapping := range p.Fields {
		index, err := p.mappingToIndex(mapping)
		if err != nil {
			return fmt.Errorf("field %v: %w", field, err)
		}

		p.fieldsIndex[field] = index
	}

	return nil
}

func (p *SwitchCase) Close() error {
	return nil
}

func (p *SwitchCase) Run() {
	for e := range p.In {
		now := time.Now()

		for label, index := range p.labelsIndex {
			oldValue, ok := e.GetLabel(label)
			if !ok {
				continue
			}

			newValue, ok := index[oldValue]
			if !ok {
				continue
			}

			e.SetLabel(label, newValue)
		}

		for field, index := range p.fieldsIndex {
			oldValueRaw, err := e.GetField(field)
			if err != nil {
				continue
			}

			oldValue, ok := oldValueRaw.(string)
			if !ok {
				p.Log.Warn(
					fmt.Sprintf("field %v exists, but it is not a string, it is a %T", field, oldValueRaw),
					elog.EventGroup(e),
				)
				continue
			}

			newValue, ok := index[oldValue]
			if !ok {
				continue
			}

			// if field exists, it can be set
			e.SetField(field, newValue)
		}

		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (p *SwitchCase) mappingToIndex(m Mapping) (map[string]string, error) {
	index := make(map[string]string, len(m))
	for newKey, oldKeys := range m {
		for _, oldKey := range oldKeys {
			if dup, exists := index[oldKey]; exists {
				return nil, fmt.Errorf("duplicate replacement found for: %v; already indexed: %v, duplicate: %v", oldKey, dup, newKey)
			}

			index[oldKey] = newKey
		}
	}
	return index, nil
}

func init() {
	plugins.AddProcessor("switchcase", func() core.Processor {
		return &SwitchCase{}
	})
}
