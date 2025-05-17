package promremote

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

// basic headers for prometheus remotewrite protocol
var basicHeaders http.Header = http.Header{
	"X-Prometheus-Remote-Write-Version": []string{"0.1.0"},
	"Content-Encoding":                  []string{"snappy"},
	"Content-Type":                      []string{"application/x-protobuf"},
}

const typicalStatsCount = 3

type Promremote struct {
	*core.BaseOutput `mapstructure:"-"`
	Host             string            `mapstructure:"host"`
	Timeout          time.Duration     `mapstructure:"timeout"`
	IdleConnTimeout  time.Duration     `mapstructure:"idle_conn_timeout"`
	IgnoreLabels     []string          `mapstructure:"ignore_labels"`
	Headerlabels     map[string]string `mapstructure:"headerlabels"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`

	client *http.Client
	ignore map[string]struct{}
}

func (o *Promremote) Init() error {
	if len(o.Host) == 0 {
		return errors.New("host required")
	}

	_, err := url.ParseRequestURI(o.Host)
	if err != nil {
		return err
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	if o.Batcher.Buffer <= 0 {
		o.Batcher.Buffer = 1
	}

	o.ignore = make(map[string]struct{}, len(o.IgnoreLabels))
	for _, v := range o.IgnoreLabels {
		o.ignore[v] = struct{}{}
	}

	o.client = &http.Client{
		Timeout: o.Timeout,
		Transport: &http.Transport{
			TLSClientConfig:   tlsConfig,
			IdleConnTimeout:   o.IdleConnTimeout,
			ForceAttemptHTTP2: tlsConfig != nil,
		},
	}

	return nil
}

func (o *Promremote) Run() {
	o.Batcher.Run(o.In, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}
		now := time.Now()

		body, err := o.marshal(buf)
		if err != nil {
			for _, e := range buf {
				o.Done <- e
				o.Log.Error("batch marshal failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				o.Observe(metrics.EventFailed, time.Since(now))
				now = time.Now()
			}
			return
		}

		header := basicHeaders.Clone()
		for k, v := range o.Headerlabels {
			if label, ok := buf[0].GetLabel(v); ok {
				header.Add(k, label)
			}
		}

		totalBefore := time.Since(now)
		now = time.Now() // reset now() to measure time spent on the request
		err = o.write(body, header)
		totalAfter := time.Since(now)

		for i, e := range buf {
			o.Done <- e
			if err != nil {
				o.Log.Error("event processing failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				o.Observe(metrics.EventFailed, durationPerEvent(totalBefore, totalAfter, len(buf), i))
			} else {
				o.Log.Debug("event processed",
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				o.Observe(metrics.EventAccepted, durationPerEvent(totalBefore, totalAfter, len(buf), i))
			}
		}
	})
}

func (o *Promremote) Close() error {
	o.client.CloseIdleConnections()
	return nil
}

func (o *Promremote) marshal(buf []*core.Event) ([]byte, error) {
	// preallocate slice with at least events count * typical stats count capacity
	series := make([]prompb.TimeSeries, 0, len(buf)*typicalStatsCount)

	for _, e := range buf {
		name, ok := e.GetLabel("::name")
		if !ok {
			o.Log.Warn("event has no ::name label, event skipped",
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			continue
		}

		rawStats, err := e.GetField("stats")
		if err != nil {
			o.Log.Warn("event has no stats field, event skipped",
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			continue
		}

		stats, ok := rawStats.(map[string]any)
		if !ok {
			o.Log.Warn("event has stats field, but it is not a map, event skipped",
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			continue
		}

		samples := make([]prompb.Sample, 0, len(stats))
		for k, v := range stats {
			val, err := convert.AnyToFloat(v)
			if err != nil {
				o.Log.Warn(fmt.Sprintf("cannot convert stats.%v to float, field skipped", k),
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				continue
			}

			samples = append(samples, prompb.Sample{
				// Timestamp for remote write should be in milliseconds
				Timestamp: e.Timestamp.UnixMilli() / int64(time.Millisecond),
				Value:     val,
			})

			labels := make([]prompb.Label, 0, len(e.Labels))
			for k, v := range e.Labels {
				// add label only if it is not in ignore list
				if _, ok := o.ignore[k]; !ok {
					labels = append(labels, prompb.Label{
						Name:  k,
						Value: v,
					})
				}
			}

			labels = append(labels, prompb.Label{
				Name:  "__name__",
				Value: name + "_" + k,
			})

			series = append(series, prompb.TimeSeries{
				Labels:  labels,
				Samples: samples,
			})
		}
	}

	if len(series) == 0 {
		return nil, errors.New("no events left after proto request preparation")
	}

	r := prompb.WriteRequest{Timeseries: series}
	body, err := r.Marshal()
	if err != nil {
		return nil, err
	}

	return snappy.Encode(nil, body), nil
}

func (o *Promremote) write(body []byte, header http.Header) error {
	return o.Retryer.Do("write metrics batch", o.Log, func() error {
		req, err := http.NewRequest(http.MethodPost, o.Host, bytes.NewReader(bytes.Clone(body)))
		if err != nil {
			return err
		}

		req.Header = header

		res, err := o.client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		rawBody, err := io.ReadAll(res.Body)
		if err != nil {
			o.Log.Warn("request performed, but body reading failed",
				"error", err,
			)
		}

		if len(rawBody) == 0 {
			rawBody = []byte("<nil>")
		}

		// this is a simple way to handle 2xx codes
		// because concrete code (e.g. 200, 201 or 204) does not specified by spec
		// https://prometheus.io/docs/specs/prw/remote_write_spec/#retries-backoff
		if res.StatusCode/100 != 2 {
			return fmt.Errorf("request result not successfull with code: %v, body: %v", res.StatusCode, string(rawBody))
		}

		return nil
	})
}

func durationPerEvent(totalBefore, totalAfter time.Duration, batchSize, i int) time.Duration {
	each := totalBefore / time.Duration(batchSize)

	if i == batchSize-1 { // last event also takes request duraion
		return each + totalAfter
	}

	return each
}

func init() {
	plugins.AddOutput("promremote", func() core.Output {
		return &Promremote{
			Timeout:         10 * time.Second,
			IdleConnTimeout: 1 * time.Minute,
			IgnoreLabels:    []string{"::type", "::name"},
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			TLSClientConfig: &tls.TLSClientConfig{},
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
