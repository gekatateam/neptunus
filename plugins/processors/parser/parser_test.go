package parser_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	"github.com/gekatateam/neptunus/plugins/processors/parser"
)

type mockParser struct {
	count int
	err   bool
}

func (m *mockParser) Parse(data []byte, routingKey string) ([]*core.Event, error) {
	if m.err {
		return nil, errors.New("what's done it's done")
	}

	var r []*core.Event
	for i := 0; i < m.count; i++ {
		r = append(r, core.NewEventWithData(
			routingKey,
			map[string]any{
				"one": "one",
				"two": i,
			},
		))
	}
	return r, nil
}

func (m *mockParser) Close() error {
	return nil
}

func (m *mockParser) Init() error {
	return nil
}

func TestParser(t *testing.T) {
	tests := map[string]*struct {
		config         map[string]any
		input          chan *core.Event
		output         chan *core.Event
		drop           chan *core.Event
		event          *core.Event
		parseCount     int
		parseError     bool
		expectFields   []map[string]any
		unexpectFields [][]string
		expectErrors   []int
	}{
		"produce-two": {
			config: map[string]any{
				"behaviour":   "produce",
				"drop_origin": true,
				"from":        "field",
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"field": "",
				},
			},
			parseCount: 2,
			parseError: false,
			expectFields: []map[string]any{
				{
					"one": "one",
					"two": 0,
				},
				{
					"one": "one",
					"two": 1,
				},
			},
			unexpectFields: [][]string{
				{},
				{},
			},
			expectErrors: []int{0, 0},
		},
		"no-field": {
			config: map[string]any{
				"behaviour":   "produce",
				"drop_origin": true,
				"from":        "field",
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"notAfield": "im not a field",
				},
			},
			parseCount: 2,
			parseError: false,
			expectFields: []map[string]any{
				{
					"notAfield": "im not a field",
				},
			},
			unexpectFields: [][]string{
				{"one", "two"},
			},
			expectErrors: []int{0},
		},
		"not-accepted-type": {
			config: map[string]any{
				"behaviour":   "produce",
				"drop_origin": true,
				"from":        "field",
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"field": 1337,
				},
			},
			parseCount: 2,
			parseError: false,
			expectFields: []map[string]any{
				{
					"field": 1337,
				},
			},
			unexpectFields: [][]string{
				{"one", "two"},
			},
			expectErrors: []int{0},
		},
		"produce-parser-error": {
			config: map[string]any{
				"behaviour":   "produce",
				"drop_origin": true,
				"from":        "field",
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"field": "im a field",
				},
			},
			parseCount: 2,
			parseError: true,
			expectFields: []map[string]any{
				{
					"field": "im a field",
				},
			},
			unexpectFields: [][]string{
				{"one", "two"},
			},
			expectErrors: []int{1},
		},
		"produce-save-origin": {
			config: map[string]any{
				"behaviour":   "produce",
				"drop_origin": false,
				"from":        "field",
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"field": "",
				},
			},
			parseCount: 2,
			parseError: false,
			expectFields: []map[string]any{
				{
					"one": "one",
					"two": 0,
				},
				{
					"one": "one",
					"two": 1,
				},
				{
					"field": "",
				},
			},
			unexpectFields: [][]string{
				{},
				{},
				{"one", "two"},
			},
			expectErrors: []int{0, 0, 0},
		},
		"merge-two-in-root": {
			config: map[string]any{
				"behaviour":   "merge",
				"drop_origin": true,
				"from":        "field",
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"field": "",
				},
			},
			parseCount: 2,
			parseError: false,
			expectFields: []map[string]any{
				{
					"one": "one",
					"two": 0,
				},
				{
					"one": "one",
					"two": 1,
				},
			},
			unexpectFields: [][]string{
				{"field"},
				{"field"},
			},
			expectErrors: []int{0, 0},
		},
		"merge-two-save-origin": {
			config: map[string]any{
				"behaviour":   "merge",
				"drop_origin": false,
				"from":        "field",
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"field": "im a field",
				},
			},
			parseCount: 2,
			parseError: false,
			expectFields: []map[string]any{
				{
					"one":   "one",
					"two":   0,
					"field": "im a field",
				},
				{
					"one":   "one",
					"two":   1,
					"field": "im a field",
				},
			},
			unexpectFields: [][]string{
				{},
				{},
			},
			expectErrors: []int{0, 0},
		},
		"merge-fail-set": {
			config: map[string]any{
				"behaviour":   "merge",
				"drop_origin": false,
				"from":        "field",
				"to":          "im.bad.way",
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"field": "im a field",
					"im":    []int{1, 2, 3},
				},
			},
			parseCount: 2,
			parseError: false,
			expectFields: []map[string]any{
				{
					"field": "im a field",
					"im":    []int{1, 2, 3},
				},
			},
			unexpectFields: [][]string{
				{"one", "two"},
			},
			expectErrors: []int{1},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			processor := &parser.Parser{
				Ider: &ider.Ider{},
				BaseProcessor: &core.BaseProcessor{
					Log: logger.Mock(),
					Obs: metrics.ObserveMock,
				},
			}
			if err := mapstructure.Decode(test.config, processor); err != nil {
				t.Fatalf("processor config not applied: %v", err)
			}
			if err := processor.Init(); err != nil {
				t.Fatalf("processor not initialized: %v", err)
			}

			wg := &sync.WaitGroup{}
			processor.SetParser(&mockParser{
				count: test.parseCount,
				err:   test.parseError,
			})
			processor.SetChannels(test.input, test.output, test.drop)
			wg.Add(1)
			go func() {
				processor.Run()
				wg.Done()
			}()

			test.event.AddHook(func() {})
			test.input <- test.event
			close(test.input)
			processor.Close()
			wg.Wait()
			close(test.output)
			close(test.drop)

			for e := range test.drop {
				e.Done()
			}

			cursor := 0
			duty := -1
			for e := range test.output {
				if len(e.Errors) != test.expectErrors[cursor] {
					t.Fatalf("expected errors: %v, got: %v; errors: %v", test.expectErrors[cursor], len(e.Errors), e.Errors)
				}

				for k, v := range test.expectFields[cursor] {
					field, err := e.GetField(k)
					if err != nil || !reflect.DeepEqual(field, v) {
						t.Fatalf("field %v, expected: %v, got: %v", k, v, field)
					}
				}

				for _, v := range test.unexpectFields[cursor] {
					field, err := e.GetField(v)
					if err == nil {
						t.Fatalf("field %v must not exists, but got %v", v, field)
					}
				}

				e.Done()
				duty = int(e.Duty())
				cursor++
			}

			if test.event.Duty() > 0 {
				t.Fatal("incoming metric not delivered")
			}

			if duty > 0 {
				t.Fatal("outgoing metric not delivered")
			}
		})
	}
}
