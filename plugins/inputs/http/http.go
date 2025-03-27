package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/netutil"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/slices"
	"github.com/gekatateam/neptunus/plugins"
	basic "github.com/gekatateam/neptunus/plugins/common/http"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	httpstats "github.com/gekatateam/neptunus/plugins/common/metrics"
	pkgtls "github.com/gekatateam/neptunus/plugins/common/tls"
)

const defaultBufferSize = 4096

type Http struct {
	*core.BaseInput `mapstructure:"-"`
	EnableMetrics   bool              `mapstructure:"enable_metrics"`
	Address         string            `mapstructure:"address"`
	Paths           []string          `mapstructure:"paths"`
	PathValues      []string          `mapstructure:"path_values"`
	ReadTimeout     time.Duration     `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration     `mapstructure:"write_timeout"`
	WaitForDelivery bool              `mapstructure:"wait_for_delivery"`
	AllowedMethods  []string          `mapstructure:"allowed_methods"`
	QueryParamsTo   string            `mapstructure:"query_params_to"`
	MaxConnections  int               `mapstructure:"max_connections"`
	LabelHeaders    map[string]string `mapstructure:"labelheaders"`

	*basic.BasicAuth        `mapstructure:",squash"`
	*ider.Ider              `mapstructure:",squash"`
	*pkgtls.TLSServerConfig `mapstructure:",squash"`

	server   *http.Server
	listener net.Listener
	parser   core.Parser

	allowedMethods map[string]struct{}
}

func (i *Http) Init() error {
	if len(i.Address) == 0 {
		return errors.New("address required")
	}

	if len(i.AllowedMethods) == 0 {
		return errors.New("at least one allowed method required")
	}

	if err := i.Ider.Init(); err != nil {
		return err
	}

	tlsConfig, err := i.TLSServerConfig.Config()
	if err != nil {
		return err
	}

	i.allowedMethods = make(map[string]struct{}, len(i.AllowedMethods))
	for _, v := range i.AllowedMethods {
		i.allowedMethods[v] = struct{}{}
	}

	var listener net.Listener
	if i.TLSServerConfig.Enable {
		l, err := tls.Listen("tcp", i.Address, tlsConfig)
		if err != nil {
			return fmt.Errorf("error creating TLS listener: %v", err)
		}
		listener = l
	} else {
		l, err := net.Listen("tcp", i.Address)
		if err != nil {
			return fmt.Errorf("error creating listener: %v", err)
		}
		listener = l
	}

	if i.MaxConnections > 0 {
		listener = netutil.LimitListener(listener, i.MaxConnections)
		i.Log.Debug(fmt.Sprintf("listener is limited to %v simultaneous connections", i.MaxConnections))
	}

	i.listener = listener
	mux := http.NewServeMux()

	var handler http.Handler = i
	if i.BasicAuth.Init() {
		handler = i.BasicAuth.Handler(handler)
	}

	if i.EnableMetrics {
		handler = httpstats.HttpServerMiddleware(i.Pipeline, i.Alias, len(i.Paths) > 0, handler)
	}

	if len(i.Paths) > 0 {
		for _, path := range slices.Unique(i.Paths) {
			mux.Handle(path, handler)
		}
		mux.Handle("/", httpstats.HttpServerMiddleware(i.Pipeline, i.Alias, true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "page not found", http.StatusNotFound)
		})))
	} else {
		mux.Handle("/", handler)
	}

	i.server = &http.Server{
		ReadTimeout:  i.ReadTimeout,
		WriteTimeout: i.WriteTimeout,
		Handler:      mux,
		TLSConfig:    tlsConfig,
		ErrorLog:     slog.NewLogLogger(i.Log.Handler(), slog.LevelError),
	}

	return nil
}

func (i *Http) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Http) Run() {
	i.Log.Info(fmt.Sprintf("starting http server on %v", i.Address))
	if err := i.server.Serve(i.listener); err != nil && err != http.ErrServerClosed {
		i.Log.Error("http server startup failed",
			"error", err.Error(),
		)
	} else {
		i.Log.Info("stopping http server")
	}
}

func (i *Http) Close() error {
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	i.server.SetKeepAlivesEnabled(false)
	if err := i.server.Shutdown(ctx); err != nil {
		i.Log.Error("http server graceful shutdown ended with error",
			"error", err.Error(),
		)
	}

	if err := i.parser.Close(); err != nil {
		i.Log.Error("parser closed with error",
			"error", err.Error(),
		)
	}

	return i.listener.Close()
}

func (i *Http) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, ok := i.allowedMethods[r.Method]; !ok {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	now := time.Now()
	wg := &sync.WaitGroup{}
	i.Log.Debug("request received",
		"sender", r.RemoteAddr,
		"path", r.URL.Path,
	)

	buf := bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		i.Log.Error("body read error",
			"error", err,
		)
		http.Error(w, fmt.Sprintf("body read error: %v", err.Error()), http.StatusInternalServerError)
		i.Observe(metrics.EventFailed, time.Since(now))
		return
	}

	var path string
	if len(i.Paths) > 0 {
		path = r.Pattern
	} else {
		path = r.URL.Path
	}

	e, err := i.parser.Parse(buf.Bytes(), path)
	if err != nil {
		i.Log.Error("parser error",
			"error", err,
		)
		http.Error(w, fmt.Sprintf("parser error: %v", err), http.StatusBadRequest)
		i.Observe(metrics.EventFailed, time.Since(now))
		return
	}

	for _, event := range e {
		event.SetLabel("server", i.Address)
		event.SetLabel("sender", r.RemoteAddr)
		event.SetLabel("method", r.Method)
		event.SetLabel("username", i.Username)

		for _, key := range i.PathValues {
			if val := r.PathValue(key); len(val) > 0 {
				event.SetLabel(key, val)
			}
		}

		for k, v := range i.LabelHeaders {
			h := r.Header.Get(v)
			if len(h) > 0 {
				event.SetLabel(k, h)
			}
		}

		if len(i.QueryParamsTo) > 0 {
			params := make(map[string]any)
			for k, v := range r.URL.Query() {
				values := make([]any, len(v))
				for i := range v {
					values[i] = v[i]
				}
				params[k] = values
			}

			if err := event.SetField(i.QueryParamsTo, params); err != nil {
				i.Log.Warn("set query params to event failed",
					"error", err,
					slog.Group("event",
						"id", event.Id,
						"key", event.RoutingKey,
					),
				)
			}
		}

		if i.WaitForDelivery {
			wg.Add(1)
			event.AddHook(wg.Done)
		}

		i.Ider.Apply(event)
		i.Out <- event
		i.Log.Debug("event accepted",
			slog.Group("event",
				"id", event.Id,
				"key", event.RoutingKey,
			),
		)
		i.Observe(metrics.EventAccepted, time.Since(now))
		now = time.Now()
	}

	wg.Wait()

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(fmt.Sprintf("accepted events: %v", len(e))))
	if err != nil {
		i.Log.Warn("all events accepted, but sending response to client failed",
			"error", err,
		)
	}
}

func init() {
	plugins.AddInput("http", func() core.Input {
		return &Http{
			Address:         ":9900",
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			MaxConnections:  0,
			AllowedMethods:  []string{"POST", "PUT"},
			BasicAuth:       &basic.BasicAuth{},
			Ider:            &ider.Ider{},
			TLSServerConfig: &pkgtls.TLSServerConfig{},
		}
	})
}
