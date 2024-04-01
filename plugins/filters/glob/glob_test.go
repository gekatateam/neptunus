package glob_test

import (
	"sync"
	"testing"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins/filters/glob"
)

func TestGlob(t *testing.T) {
	tests := map[string]struct {
		config         map[string]any
		input          chan *core.Event
		accept         chan *core.Event
		reject         chan *core.Event
		events         []*core.Event
		expectedAccept int
		expectedReject int
	}{
		"all-must-pass-with-empty-cfg": {
			config: map[string]any{},
			input:  make(chan *core.Event, 100),
			accept: make(chan *core.Event, 100),
			reject: make(chan *core.Event, 100),
			events: []*core.Event{
				core.NewEvent("rk1"),
				core.NewEvent("rk1"),
			},
			expectedAccept: 2,
			expectedReject: 0,
		},
		"must-split-by-routing-key": {
			config: map[string]any{
				"routing_key": []string{"pass-me", "passed-*-key", "pass-me-to"},
			},
			input:  make(chan *core.Event, 100),
			accept: make(chan *core.Event, 100),
			reject: make(chan *core.Event, 100),
			events: []*core.Event{
				core.NewEvent("passed-test-key"),
				core.NewEvent("rejected-key"),
			},
			expectedAccept: 1,
			expectedReject: 1,
		},
		"must-split-by-key-and-field": {
			config: map[string]any{
				"routing_key": []string{"pass-me", "passed-*-key", "pass-me-to"},
				"fields": map[string][]string{
					"one.two": {"t*ee"},
				},
			},
			input:  make(chan *core.Event, 100),
			accept: make(chan *core.Event, 100),
			reject: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					RoutingKey: "passed-test-key",
					Data: map[string]any{
						"one": map[string]any{
							"two": "three",
						},
						"three": map[string]any{
							"two": "three",
						},
					},
				},
				{
					RoutingKey: "passed-test-key",
					Data: map[string]any{
						"one": map[string]any{
							"two": "four",
						},
						"three": map[string]any{
							"two": "three",
						},
					},
				},
			},
			expectedAccept: 1,
			expectedReject: 1,
		},
		"must-split-by-label-and-field": {
			config: map[string]any{
				"fields": map[string][]string{
					"one.two": {"t*ee"},
				},
				"labels": map[string][]string{
					"test": {"t*e"},
				},
			},
			input:  make(chan *core.Event, 100),
			accept: make(chan *core.Event, 100),
			reject: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					RoutingKey: "passed-test-key",
					Labels: map[string]string{
						"test": "true",
					},
					Data: map[string]any{
						"one": map[string]any{
							"two": "three",
						},
						"three": map[string]any{
							"two": "three",
						},
					},
				},
				{
					RoutingKey: "passed-test-key",
					Labels: map[string]string{
						"test": "false",
					},
					Data: map[string]any{
						"one": map[string]any{
							"two": "four",
						},
						"three": map[string]any{
							"two": "three",
						},
					},
				},
			},
			expectedAccept: 1,
			expectedReject: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			filter := &glob.Glob{
				BaseFilter: &core.BaseFilter{
					Log: logger.Mock(),
					Obs: metrics.ObserveMock,
				},
			}
			if err := mapstructure.Decode(test.config, filter); err != nil {
				t.Fatalf("filter config not applied: %v", err)
			}
			if err := filter.Init(); err != nil {
				t.Fatalf("filter not initialized: %v", err)
			}

			wg := &sync.WaitGroup{}
			filter.SetChannels(test.input, test.reject, test.accept)
			wg.Add(1)
			go func() {
				filter.Run()
				wg.Done()
			}()

			for _, e := range test.events {
				test.input <- e
			}

			close(test.input)
			filter.Close()
			wg.Wait()

			if len(test.accept) != test.expectedAccept {
				t.Fatalf("unexpected accepted messages count - want: %v, got: %v", test.expectedAccept, len(test.accept))
			}

			if len(test.reject) != test.expectedReject {
				t.Fatalf("unexpected rejected messages count - want: %v, got: %v", test.expectedReject, len(test.reject))
			}
		})
	}
}
