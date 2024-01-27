package stats_test

import (
	"hash/fnv"
	"sync"
	"testing"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins/processors/stats"
)

func TestStats(t *testing.T) {
	tests := map[string]struct {
		config       map[string]any
		input        chan *core.Event
		output       chan *core.Event
		events       []*core.Event
		expectEvents map[uint64]map[string]float64
	}{
		"sum-count-gauge-test": {
			config: map[string]any{
				"period":      "1m",
				"routing_key": "ngm",
				"labels":      []string{"code", "proto"},
				"drop_origin": true, // in all tests processor drops origin events
				"mode":        "individual",
				"fields": map[string][]string{
					"duration": {"count", "sum", "gauge"},
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: core.Map{
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
					Data: core.Map{
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
					Data: core.Map{
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
				"labels":      []string{"code", "proto"},
				"drop_origin": true, // in all tests processor drops origin events
				"mode":        "individual",
				"fields": map[string][]string{
					"duration": {"count", "sum", "gauge"},
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: core.Map{
						"duration": 11,
						"uri":      "/users/111",
					},
				},
				{ // this event has no proto label, it must be ignored
					Labels: map[string]string{
						"code":   "200",
						"client": "1.2.3.4",
					},
					Data: core.Map{
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
					Data: core.Map{
						"uri": "/users/111",
					},
				},
				{ // this event has NaN duration field type, it must be ignored
					Labels: map[string]string{
						"code":   "500",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: core.Map{
						"duration": "45",
						"uri":      "/users/111",
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
				"labels":      []string{"code", "proto"},
				"drop_origin": true, // in all tests processor drops origin events
				"mode":        "individual",
				"fields": map[string][]string{
					"duration": {"min", "max", "avg"},
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: core.Map{
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
					Data: core.Map{
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
					Data: core.Map{
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
				"labels":      []string{"code", "proto"},
				"drop_origin": true, // in all tests processor drops origin events
				"mode":        "individual",
				"fields": map[string][]string{
					"duration":      {"sum"},
					"response_time": {"avg"},
				},
			},
			input:  make(chan *core.Event, 100),
			output: make(chan *core.Event, 100),
			events: []*core.Event{
				{
					Labels: map[string]string{
						"code":   "200",
						"proto":  "HTTP/1.0",
						"client": "1.2.3.4",
					},
					Data: core.Map{
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
					Data: core.Map{
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

			for e := range test.output {
				labels := []label{}
				for _, v := range test.config["labels"].([]string) {
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
