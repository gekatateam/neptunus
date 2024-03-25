package regex_test

import (
	"sync"
	"testing"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins/processors/regex"
)

func TestRegex(t *testing.T) {
	tests := map[string]struct {
		config         map[string]any
		input          chan *core.Event
		output         chan *core.Event
		events         []*core.Event
		expectLabels   []map[string]string
		expectFields   []map[string]string
		unexpectFields [][]string
		expectErrors   []int
	}{
		"no-regex": {
			config: map[string]any{},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					RoutingKey: "rk",
					Data: core.Map{
						"one": core.Map{
							"two": "one: data1 two: 1337 [leet info]",
						},
					},
				},
			},
			expectLabels: []map[string]string{
				{},
			},
			expectFields: []map[string]string{
				{
					"one.two": "one: data1 two: 1337 [leet info]",
				},
			},
			unexpectFields: [][]string{
				{},
			},
			expectErrors: []int{0},
		},
		"labels-regex-ok": {
			config: map[string]any{
				"labels": map[string]string{
					"testlabelone": `^first:(?P<first>[a-z]+)\ssecond:(?P<second>[a-z]+)$`,
					"testlabeltwo": `^foo\s(?P<foo>[a-z]+)\sfizz\s(?P<fizz>\d+)$`,
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					RoutingKey: "rk",
					Data:       core.Map{},
					Labels: map[string]string{
						"testlabelone": "first:f second:sec",
					},
				},
				{
					RoutingKey: "rk",
					Data:       core.Map{},
					Labels: map[string]string{
						"testlabelone": "first:f second:sec",
						"testlabeltwo": "foo bar fizz 1234",
					},
				},
			},
			expectLabels: []map[string]string{
				{
					"first":  "f",
					"second": "sec",
				},
				{
					"first":  "f",
					"second": "sec",
					"foo":    "bar",
					"fizz":   "1234",
				},
			},
			expectFields: []map[string]string{
				{},
				{},
			},
			unexpectFields: [][]string{
				{},
				{},
			},
			expectErrors: []int{
				0,
				0,
			},
		},
		"fields-regex-ok": {
			config: map[string]any{
				"fields": map[string]string{
					"test.one": `^first:(?P<first>[a-z]+)\ssecond:(?P<second>[a-z]+)$`,
					"test.two": `^foo\s(?P<foo>[a-z]+)\sfizz\s(?P<fizz>\d+)$`,
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					RoutingKey: "rk",
					Data: core.Map{
						"test": core.Map{
							"one": "first:f second:sec",
							"two": "foo bar fizz 1234",
						},
					},
				},
				{
					RoutingKey: "rk",
					Data: core.Map{
						"test": core.Map{
							"one": "first:f second:sec",
							"two": "foo bar but must not match",
						},
					},
				},
			},
			expectLabels: []map[string]string{
				{},
				{},
			},
			expectFields: []map[string]string{
				{
					"first":  "f",
					"second": "sec",
					"foo":    "bar",
					"fizz":   "1234",
				},
				{
					"first":  "f",
					"second": "sec",
				},
			},
			unexpectFields: [][]string{
				{},
				{"foo", "fizz"},
			},
			expectErrors: []int{
				0,
				0,
			},
		},
		"fields-regex-overwrite": {
			config: map[string]any{
				"fields": map[string]string{
					"test.one": `^first:(?P<first>[a-z]+)\ssecond:(?P<second>[a-z]+)$`,
					"test.two": `^foo\s(?P<foo>[a-z]+)\sfizz\s(?P<fizz>\d+)$`,
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					RoutingKey: "rk",
					Data: core.Map{
						"test": core.Map{
							"two": "foo bar fizz 1234",
						},
						"fizz": []int{0, 10, 100},
					},
				},
			},
			expectLabels: []map[string]string{
				{},
			},
			expectFields: []map[string]string{
				{
					"foo":  "bar",
					"fizz": "1234",
				},
			},
			unexpectFields: [][]string{
				{},
			},
			expectErrors: []int{0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			processor := &regex.Regex{
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
			processor.SetChannels(test.input, test.output)
			wg.Add(1)
			go func() {
				processor.Run()
				wg.Done()
			}()

			for _, e := range test.events {
				test.input <- e
			}
			close(test.input)
			processor.Close()
			wg.Wait()
			close(test.output)

			cursor := 0
			for e := range test.output {
				if len(e.Errors) != test.expectErrors[cursor] {
					t.Fatalf("expected errors: %v, got: %v; errors: %v", test.expectErrors[cursor], len(e.Errors), e.Errors)
				}

				for k, v := range test.expectLabels[cursor] {
					label, ok := e.GetLabel(k)
					if !ok || label != v {
						t.Fatalf("label %v, expected: %v, got: %v", k, v, label)
					}
				}

				for k, v := range test.expectFields[cursor] {
					field, err := e.GetField(k)
					if err != nil || field.(string) != v {
						t.Fatalf("field %v, expected: %v, got: %v", k, v, field)
					}
				}

				for _, v := range test.unexpectFields[cursor] {
					field, err := e.GetField(v)
					if err == nil {
						t.Fatalf("field %v must not exists, but got %v", v, field)
					}
				}

				cursor++
			}
		})
	}
}
