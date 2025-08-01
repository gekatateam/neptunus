package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

type requester struct {
	*core.BaseOutput

	uri         *url.URL
	fallbacks   []*url.URL
	method      string
	methodLabel string

	successBody  *regexp.Regexp
	successCodes map[int]struct{}
	headerlabels map[string]string
	paramfields  map[string]string

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

		header := make(http.Header)
		for k, v := range r.headerlabels {
			if label, ok := buf[0].GetLabel(v); ok {
				header.Add(k, label)
			}
		}

		values, err := r.unpackQueryValues(buf[0])
		if err != nil {
			each := time.Since(now) / time.Duration(len(buf))
			for _, e := range buf {
				r.Log.Error("query params prepation failed",
					"error", err,
					elog.EventGroup(e),
				)
				r.Done <- e
				r.Observe(metrics.EventFailed, each)
			}
			return
		}

		rawBody, err := r.ser.Serialize(buf...)
		if err != nil {
			each := time.Since(now) / time.Duration(len(buf))
			for _, e := range buf {
				r.Log.Error("event serialization failed",
					"error", err,
					elog.EventGroup(e),
				)
				r.Done <- e
				r.Observe(metrics.EventFailed, each)
			}
			return
		}

		totalBefore := time.Since(now)
		now = time.Now() // reset now() to measure time spent on the request
		err = r.performWithFallback(r.requestMethod(buf[0]), values, rawBody, header)
		totalAfter := time.Since(now)

		for i, e := range buf {
			r.Done <- e
			if err != nil {
				r.Log.Error("event processing failed",
					"error", err,
					elog.EventGroup(e),
				)
				r.Observe(metrics.EventFailed, durationPerEvent(totalBefore, totalAfter, len(buf), i))
			} else {
				r.Log.Debug("event processed",
					elog.EventGroup(e),
				)
				r.Observe(metrics.EventAccepted, durationPerEvent(totalBefore, totalAfter, len(buf), i))
			}
		}
	})

	r.Log.Info(fmt.Sprintf("requester for %v closed", r.uri))
}

func (r *requester) Push(e *core.Event) {
	r.input <- e
}

func (r *requester) Close() error {
	close(r.input)
	return nil
}

func (r *requester) unpackQueryValues(e *core.Event) (url.Values, error) {
	values := make(url.Values)

	for k, v := range r.paramfields {
		field, err := e.GetField(v)
		if err != nil {
			continue // skip if target field not found
		}

		switch f := field.(type) {
		case []any:
			for i, j := range f {
				param, err := convert.AnyToString(j)
				if err != nil {
					return nil, fmt.Errorf("%v.%v: %w", v, i, err)
				}

				values.Add(k, param)
			}
		default:
			param, err := convert.AnyToString(f)
			if err != nil {
				return nil, fmt.Errorf("%v: %w", v, err)
			}

			values.Add(k, param)
		}
	}

	return values, nil
}

func (r *requester) performWithFallback(method string, params url.Values, body []byte, header http.Header) error {
	err := r.perform(r.uri, method, params, body, header)

	if err != nil && len(r.fallbacks) > 0 {
		r.Log.Warn("request failed, trying to perform to fallback",
			"error", err,
		)

		for _, f := range r.fallbacks {
			err = r.perform(f, method, params, body, header)
			if err == nil {
				r.Log.Warn(fmt.Sprintf("request to %v succeeded, but it is still a fallback", f.String()))
				return nil
			}

			r.Log.Warn(fmt.Sprintf("request to fallback %v failed", f.String()),
				"error", err,
			)
		}
	}

	return err
}

func (r *requester) perform(uri *url.URL, method string, params url.Values, body []byte, header http.Header) error {
	uri.RawQuery = params.Encode()

	return r.Retryer.Do("perform request", r.Log, func() error {
		req, err := http.NewRequest(method, uri.String(), bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header = header.Clone()

		r.Log.Debug(fmt.Sprintf("request body: %v; request headers: %v; request query: %v", string(body), header, uri.RawQuery))

		res, err := r.client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		rawBody, err := io.ReadAll(res.Body)
		if err != nil {
			r.Log.Warn("request performed, but body reading failed",
				"error", err,
			)
		}

		if _, ok := r.successCodes[res.StatusCode]; ok {
			r.Log.Debug(fmt.Sprintf("request performed successfully with code: %v, body: %v", res.StatusCode, string(rawBody)))
			return nil
		} else {
			if r.successBody != nil && r.successBody.Match(rawBody) {
				r.Log.Debug(fmt.Sprintf("request performed with code: %v, body: %v; "+
					"but response body matches configured regexp", res.StatusCode, string(rawBody)))
				return nil
			}

			return fmt.Errorf("request result not successful with code: %v, body: %v", res.StatusCode, string(rawBody))
		}
	})
}

func (r *requester) requestMethod(e *core.Event) string {
	if l := r.methodLabel; len(l) > 0 {
		if method, ok := e.GetLabel(l); ok && len(method) > 0 {
			return method
		}
		r.Log.Warn("event has no label with request method or it's empty")
	}
	return r.method
}

func durationPerEvent(totalBefore, totalAfter time.Duration, batchSize, i int) time.Duration {
	each := totalBefore / time.Duration(batchSize)

	if i == batchSize-1 { // last event also takes request duraion
		return each + totalAfter
	}

	return each
}
