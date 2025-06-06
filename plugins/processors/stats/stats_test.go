package stats_test

import (
	"hash/fnv"
	"sync"
	"testing"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins/processors/stats"
)

func TestStats_WithLabelsConfigured(t *testing.T) {
	tests := map[string]struct {
		config       map[string]any
		input        chan *core.Event
		output       chan *core.Event
		drop         chan *core.Event
		events       []*core.Event
		expectEvents map[uint64]map[string]float64
	}{
		"sum-count-gauge-test": {
			config: map[string]any{
				"period":      "1m",
				"routing_key": "ngm",
				"with_labels": []string{"code", "proto"},
				"drop_origin": true, // in all tests processor drops origin events
				"mode":        "individual",
				"fields": map[string][]string{
					"duration": {"count", "sum", "gauge"},
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 11,
						"uri":      "/users/111",
					},
				},
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 22,
						"uri":      "/users/111",
					},
				},
				{
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 22,
						"uri":      "/users/111",
					},
				},
			},
			expectEvents: map[uint64]map[string]float64{
				7803659511733462090: {
					"stats.count": 2,
					"stats.sum":   33,
					"stats.gauge": 22,
				},
				7003399780367318885: {
					"stats.count": 1,
					"stats.sum":   22,
					"stats.gauge": 22,
				},
			},
		},
		"no-label-or-field-test": {
			config: map[string]any{
				"period":      "1m",
				"routing_key": "ngm",
				"with_labels": []string{"code", "proto"},
				"drop_origin": true, // in all tests processor drops origin events
				"mode":        "individual",
				"fields": map[string][]string{
					"duration": {"count", "sum", "gauge"},
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 11,
						"uri":      "/users/111",
					},
				},
				{ // this event has no proto label, it must be ignored
					Labels: map[string]string{
						"code":   "200",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 22,
						"uri":      "/users/111",
					},
				},
				{ // this event has no duration field, it must be ignored
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"uri": "/users/111",
					},
				},
			},
			expectEvents: map[uint64]map[string]float64{
				7803659511733462090: {
					"stats.count": 1,
					"stats.sum":   11,
					"stats.gauge": 11,
				},
			},
		},
		"min-max-avg-test": {
			config: map[string]any{
				"period":      "1m",
				"routing_key": "ngm",
				"with_labels": []string{"code", "proto"},
				"drop_origin": true, // in all tests processor drops origin events
				"mode":        "individual",
				"fields": map[string][]string{
					"duration": {"min", "max", "avg"},
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 11,
						"uri":      "/users/111",
					},
				},
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 22,
						"uri":      "/users/111",
					},
				},
				{
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 22,
						"uri":      "/users/111",
					},
				},
			},
			expectEvents: map[uint64]map[string]float64{
				7803659511733462090: {
					"stats.min": 11,
					"stats.max": 22,
					"stats.avg": 16.5,
				},
				7003399780367318885: {
					"stats.min": 22,
					"stats.max": 22,
					"stats.avg": 22,
				},
			},
		},
		"multiple-fields-test": {
			config: map[string]any{
				"period":      "1m",
				"routing_key": "ngm",
				"with_labels": []string{"code", "proto"},
				"drop_origin": true, // in all tests processor drops origin events
				"mode":        "individual",
				"fields": map[string][]string{
					"duration":      {"sum"},
					"response_time": {"avg"},
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			drop:   make(chan *core.Event, 100),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration":      11,
						"response_time": 22,
						"uri":           "/users/111",
					},
				},
				{
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration":      44,
						"response_time": 33,
						"uri":           "/users/111",
					},
				},
			},
			expectEvents: map[uint64]map[string]float64{
				7803659511733462090: {
					"stats.sum": 11,
				},
				7003399780367318885: {
					"stats.sum": 44,
				},
				12761526936774808649: {
					"stats.avg": 22,
				},
				10242469807971764870: {
					"stats.avg": 33,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			processor := &stats.Stats{
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
			processor.SetChannels(test.input, test.output, test.drop)
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
			close(test.drop)
			close(test.output)

			for e := range test.drop {
				e.Done()
			}

			for e := range test.output {
				labels := []label{}
				for _, v := range test.config["with_labels"].([]string) {
					l, ok := e.GetLabel(v)
					if !ok {
						t.Fatalf("stats event has no expected label: %v", v)
					}
					labels = append(labels, label{
						Key:   v,
						Value: l,
					})
				}

				name, ok := e.GetLabel("::name")
				if !ok {
					t.Fatal("stats event has no ::name label")
				}

				h := hash(name, labels)
				s, ok := test.expectEvents[h]
				if !ok {
					t.Fatalf("stats event was not found in expected events: %v", *e)
				}

				for f, v := range s {
					val, err := e.GetField(f)
					if err != nil {
						t.Fatalf("stats event has no field %v", f)
					}

					if v != val.(float64) {
						t.Fatalf("unexpected value for field %v; got: %v, want: %v", f, val, v)
					}
				}

				delete(test.expectEvents, h)
			}

			if len(test.expectEvents) != 0 {
				t.Fatalf("processor not produced all expexted events, lost: %v", test.expectEvents)
			}

			if len(test.output) != 0 {
				t.Fatalf("processor produced more than expexted events, left: %v", len(test.output))
			}
		})
	}
}

// generated by copilot
func TestStats_NoLabelsConfigured(t *testing.T) {
	tests := map[string]struct {
		config       map[string]any
		input        chan *core.Event
		output       chan *core.Event
		drop         chan *core.Event
		events       []*core.Event
		expectEvents map[uint64]map[string]float64
	}{
		"no-labels-configured-count-sum": {
			config: map[string]any{
				"period":      "1m",
				"routing_key": "ngm",
				"drop_origin": true,
				"mode":        "individual",
				"fields": map[string][]string{
					"duration": {"count", "sum"},
				},
			},
			input:  make(chan *core.Event, 10),
			output: make(chan *core.Event, 10),
			drop:   make(chan *core.Event, 10),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 10,
					},
				},
				{
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 20,
					},
				},
			},
			expectEvents: map[uint64]map[string]float64{
				hash("duration", nil): {
					"stats.count": 2,
					"stats.sum":   30,
				},
			},
		},
		"no-labels-configured-min-max": {
			config: map[string]any{
				"period":      "1m",
				"routing_key": "ngm",
				"drop_origin": true,
				"mode":        "individual",
				"fields": map[string][]string{
					"duration": {"min", "max"},
				},
			},
			input:  make(chan *core.Event, 10),
			output: make(chan *core.Event, 10),
			drop:   make(chan *core.Event, 10),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 5,
					},
				},
				{
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 15,
					},
				},
			},
			expectEvents: map[uint64]map[string]float64{
				hash("duration", nil): {
					"stats.min": 5,
					"stats.max": 15,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			processor := &stats.Stats{
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
			processor.SetChannels(test.input, test.output, test.drop)
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
			close(test.drop)
			close(test.output)

			for e := range test.drop {
				e.Done()
			}

			for e := range test.output {
				name, ok := e.GetLabel("::name")
				if !ok {
					t.Fatal("stats event has no ::name label")
				}
				// No labels configured, so pass nil
				h := hash(name, nil)
				s, ok := test.expectEvents[h]
				if !ok {
					t.Fatalf("stats event was not found in expected events: %v", *e)
				}

				for f, v := range s {
					val, err := e.GetField(f)
					if err != nil {
						t.Fatalf("stats event has no field %v", f)
					}
					if v != val.(float64) {
						t.Fatalf("unexpected value for field %v; got: %v, want: %v", f, val, v)
					}
				}
				delete(test.expectEvents, h)
			}

			if len(test.expectEvents) != 0 {
				t.Fatalf("processor not produced all expected events, lost: %v", test.expectEvents)
			}
			if len(test.output) != 0 {
				t.Fatalf("processor produced more than expected events, left: %v", len(test.output))
			}
		})
	}
}

// generated by copilot
func TestStats_WithoutLabelsConfigured(t *testing.T) {
	tests := map[string]struct {
		config       map[string]any
		input        chan *core.Event
		output       chan *core.Event
		drop         chan *core.Event
		events       []*core.Event
		expectEvents map[uint64]map[string]float64
	}{
		"without-labels-count-sum": {
			config: map[string]any{
				"period":         "1m",
				"routing_key":    "ngm",
				"without_labels": []string{"client"},
				"drop_origin":    true,
				"mode":           "individual",
				"fields": map[string][]string{
					"duration": {"count", "sum"},
				},
			},
			input:  make(chan *core.Event, 10),
			output: make(chan *core.Event, 10),
			drop:   make(chan *core.Event, 10),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 10,
					},
				},
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "5.6.7.8",
					},
					Data: map[string]any{
						"duration": 20,
					},
				},
				{
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/2.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 30,
					},
				},
			},
			expectEvents: map[uint64]map[string]float64{
				hash("duration", []label{
					{Key: "code", Value: "200"},
					{Key: "proto", Value: "HTTP/1.0"},
				}): {
					"stats.count": 2,
					"stats.sum":   30,
				},
				hash("duration", []label{
					{Key: "code", Value: "500"},
					{Key: "proto", Value: "HTTP/2.0"},
				}): {
					"stats.count": 1,
					"stats.sum":   30,
				},
			},
		},
		"without-labels-min-max": {
			config: map[string]any{
				"period":         "1m",
				"routing_key":    "ngm",
				"without_labels": []string{"client"},
				"drop_origin":    true,
				"mode":           "individual",
				"fields": map[string][]string{
					"duration": {"min", "max"},
				},
			},
			input:  make(chan *core.Event, 10),
			output: make(chan *core.Event, 10),
			drop:   make(chan *core.Event, 10),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 5,
					},
				},
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "5.6.7.8",
					},
					Data: map[string]any{
						"duration": 15,
					},
				},
				{
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/2.0",
						"client": "1.2.3.4",
					},
					Data: map[string]any{
						"duration": 25,
					},
				},
			},
			expectEvents: map[uint64]map[string]float64{
				hash("duration", []label{
					{Key: "code", Value: "200"},
					{Key: "proto", Value: "HTTP/1.0"},
				}): {
					"stats.min": 5,
					"stats.max": 15,
				},
				hash("duration", []label{
					{Key: "code", Value: "500"},
					{Key: "proto", Value: "HTTP/2.0"},
				}): {
					"stats.min": 25,
					"stats.max": 25,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			processor := &stats.Stats{
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
			processor.SetChannels(test.input, test.output, test.drop)
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
			close(test.drop)
			close(test.output)

			for e := range test.drop {
				e.Done()
			}

			for e := range test.output {
				// Collect all labels except "client" (since it's excluded)
				var labels []label
				for k, v := range e.Labels {
					if k == "::type" || k == "::name" || k == "client" {
						continue
					}
					labels = append(labels, label{Key: k, Value: v})
				}
				// Sort labels for deterministic hash
				if len(labels) > 1 && labels[0].Key > labels[1].Key {
					labels[0], labels[1] = labels[1], labels[0]
				}
				name, ok := e.GetLabel("::name")
				if !ok {
					t.Fatal("stats event has no ::name label")
				}
				h := hash(name, labels)
				s, ok := test.expectEvents[h]
				if !ok {
					t.Fatalf("stats event was not found in expected events: %v", *e)
				}
				for f, v := range s {
					val, err := e.GetField(f)
					if err != nil {
						t.Fatalf("stats event has no field %v", f)
					}
					if v != val.(float64) {
						t.Fatalf("unexpected value for field %v; got: %v, want: %v", f, val, v)
					}
				}
				delete(test.expectEvents, h)
			}

			if len(test.expectEvents) != 0 {
				t.Fatalf("processor not produced all expected events, lost: %v", test.expectEvents)
			}
			if len(test.output) != 0 {
				t.Fatalf("processor produced more than expected events, left: %v", len(test.output))
			}
		})
	}
}

type label struct {
	Key   string
	Value string
}

func hash(name string, labels []label) uint64 {
	h := fnv.New64a()
	h.Write([]byte(name))
	for _, v := range labels {
		h.Write([]byte(v.Key))
		h.Write([]byte(v.Value))
	}
	return h.Sum64()
}
