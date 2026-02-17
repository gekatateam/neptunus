package csv

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

const (
	ModeHorizontal = "horizontal"
	ModeVertical   = "vertical"
)

type Csv struct {
	*core.BaseParser `mapstructure:"-"`
	Mode             string `mapstructure:"mode"`
	KeyColumn        string `mapstructure:"key_column"`
	HasHeader        bool   `mapstructure:"has_header"`
	LazyQuotes       bool   `mapstructure:"lazy_quotes"`
	Comma            rune   `mapstructure:"comma"`
	Comment          rune   `mapstructure:"comment"`
}

func (p *Csv) Init() error {
	switch p.Mode {
	case ModeHorizontal:
	case ModeVertical:
		if len(p.KeyColumn) == 0 {
			return errors.New("key column required")
		}

		if !p.HasHeader {
			return errors.New("header required for vertical mode")
		}
	default:
		return fmt.Errorf("unknown mode: %v", p.Mode)
	}

	return nil
}

func (p *Csv) Close() error {
	return nil
}

func (p *Csv) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	now := time.Now()

	var (
		events []*core.Event
		err    error
	)

	switch p.Mode {
	case ModeHorizontal:
		events, err = p.parseHorizontal(data, routingKey)
	// case ModeVertical:
	// 	events, err = p.parseVertical(data, routingKey)
	default:
		panic(fmt.Errorf("totally unexpected - how you start it with this mode? %v", p.Mode))
	}

	if err != nil {
		p.Observe(metrics.EventFailed, time.Since(now))
		return nil, err
	}

	for range events {
		p.Observe(metrics.EventAccepted, time.Since(now))
		now = time.Now()
	}

	return events, nil
}

func (p *Csv) parseHorizontal(data []byte, routingKey string) ([]*core.Event, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = p.Comma
	reader.Comment = p.Comment
	reader.LazyQuotes = p.LazyQuotes
	reader.ReuseRecord = true

	var events []*core.Event
	if p.HasHeader {
		headerRaw, err := reader.Read()
		if err == io.EOF { // zero events if no data provided
			return events, nil
		}

		if err != nil {
			return nil, err
		}

		header := slices.Clone(headerRaw)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}

			if err != nil {
				return nil, err
			}

			body := make(map[string]any, len(header))
			for i, h := range header {
				body[h] = record[i]
			}
			events = append(events, core.NewEventWithData(routingKey, body))
		}
	} else {
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}

			if err != nil {
				return nil, err
			}

			body := make([]any, len(record))
			for i, v := range record {
				body[i] = v
			}
			events = append(events, core.NewEventWithData(routingKey, body))
		}
	}

	return events, nil
}

func init() {
	plugins.AddParser("csv", func() core.Parser {
		return &Csv{
			Mode:       ModeHorizontal,
			HasHeader:  true,
			Comma:      ';',
			Comment:    0,
			LazyQuotes: false,
		}
	})
}
