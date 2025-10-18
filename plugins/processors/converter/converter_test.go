package converter_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins/processors/converter"
)

func TestConverter(t *testing.T) {
	tests := map[string]struct {
		config       map[string]any
		input        chan *core.Event
		output       chan *core.Event
		event        *core.Event
		expectedData any
		expectErrors bool
	}{
		"convert-to-string-ok": {
			config: map[string]any{
				"string": []string{
					"field:string", "field:float", "field:int", "field:uint", "field:bool", "label:foo",
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			event: &core.Event{
				Labels: map[string]string{
					"foo": "bar",
				},
				Data: map[string]any{
					"string": "string field",
					"float":  13.133337,
					"int":    -200,
					"uint":   504,
					"bool":   true,
				},
			},
			expectedData: map[string]any{
				"string": "string field",
				"float":  "13.133337",
				"int":    "-200",
				"uint":   "504",
				"bool":   "true",
				"foo":    "bar",
			},
			expectErrors: false,
		},
		"convert-to-int-ok": {
			config: map[string]any{
				"integer": []string{
					"field:string", "field:float", "field:int", "field:uint", "field:bool", "label:foo",
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			event: &core.Event{
				Labels: map[string]string{
					"foo": "99",
				},
				Data: map[string]any{
					"string": "1337",
					"float":  13.133337,
					"int":    -200,
					"uint":   504,
					"bool":   true,
				},
			},
			expectedData: map[string]any{
				"string": int64(1337),
				"float":  int64(13),
				"int":    int64(-200),
				"uint":   int64(504),
				"bool":   int64(1),
				"foo":    int64(99),
			},
			expectErrors: false,
		},
		"convert-to-uint-ignore-overflow-ok": {
			config: map[string]any{
				"ignore_out_of_range": true,
				"unsigned": []string{
					"field:string", "field:float", "field:int", "field:uint", "field:bool", "label:foo",
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			event: &core.Event{
				Labels: map[string]string{
					"foo": "99",
				},
				Data: map[string]any{
					"string": "1337",
					"float":  13.133337,
					"int":    -200,
					"uint":   uint(504),
					"bool":   true,
				},
			},
			expectedData: map[string]any{
				"string": uint64(1337),
				"float":  uint64(13),
				"int":    uint64(18446744073709551416), // expected behaviour btw
				"uint":   uint64(504),
				"bool":   uint64(1),
				"foo":    uint64(99),
			},
			expectErrors: false,
		},
		"convert-to-uint-overflow-err": {
			config: map[string]any{
				"ignore_overflow": false,
				"unsigned":        []string{"field:int"},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			event: &core.Event{
				Data: map[string]any{
					"int": -200,
				},
			},
			expectedData: map[string]any{
				"int": -200,
			},
			expectErrors: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			processor := &converter.Converter{
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
			processor.SetChannels(test.input, test.output, nil)
			wg.Add(1)
			go func() {
				processor.Run()
				wg.Done()
			}()

			test.input <- test.event
			close(test.input)
			processor.Close()
			wg.Wait()
			close(test.output)
			e := <-test.output

			if test.expectErrors != (len(e.Errors) > 0) {
				t.Errorf("errors condition failed: %v", e.Errors)
			}

			if !reflect.DeepEqual(test.expectedData, e.Data) {
				t.Errorf("unexpected result, want: %#v, got: %#v", test.expectedData, e.Data)
			}
		})
	}
}
