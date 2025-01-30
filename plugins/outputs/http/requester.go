package http

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

type requester struct {
	*core.BaseOutput

	lastWrite time.Time
	uri       string
	method    string

	retryableCodes map[int]struct{}
	headerlabels   map[string]string
	paramfields    map[string]string

	client *http.Client
	*batcher.Batcher[*core.Event]
	*retryer.Retryer

	ser   core.Serializer
	input chan *core.Event
}

func (r *requester) Run() {
	r.Log.Info(fmt.Sprintf("requester for %v spawned", r.uri))

	r.Batcher.Run(r.input, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}
		now := time.Now()
		r.lastWrite = now

		headers := map[string]string{}
		for k, v := range r.headerlabels {
			if label, ok := buf[0].GetLabel(v); ok {
				headers[k] = label
			}
		}

		rawBody, err := r.ser.Serialize(buf...)
		t := durationPerEvent(time.Since(now), len(buf))
		if err != nil {
			for _, e := range buf {
				r.Log.Error("event serialization failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				r.Done <- e
				r.Observe(metrics.EventFailed, t)
			}
			return
		}

		now = time.Now() // reset now() to correctly measure the time spent on the query
		err = r.perform(rawBody, headers)
		ts := time.Since(now)

		if err != nil {
			for i, e := range buf {
				r.Log.Error("event processing failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				r.Done <- e
				r.Observe(metrics.EventFailed, durationLastEvent(i, len(buf), t, ts))
			}
		} else {
			for i, e := range buf {
				r.Log.Debug("event processed",
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				r.Done <- e
				r.Observe(metrics.EventAccepted, durationLastEvent(i, len(buf), t, ts))
			}
		}
	})

	r.Log.Info(fmt.Sprintf("requester for %v closed", r.uri))
}

func (r *requester) Push(e *core.Event) {
	r.input <- e
}

func (r *requester) LastWrite() time.Time {
	return r.lastWrite
}

func (r *requester) Close() error {
	close(r.input)
	return nil
}

func (r *requester) perform(uri string, params url.Values, body []byte, headers map[string]string) error {
	req, err := http.NewRequest(r.method, r.url.String(), bytes.NewReader(bytes.Clone(body)))

}

func durationPerEvent(totalTime time.Duration, batchSize int) time.Duration {
	return time.Duration(int64(totalTime) / int64(batchSize))
}

func durationLastEvent(i, len int, t1, t2 time.Duration) time.Duration {
	if i == len-1 {
		return t1 + t2
	}
	return t1
}
