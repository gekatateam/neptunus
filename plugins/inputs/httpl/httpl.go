package httpl

import (
	"bufio"
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
	"github.com/gekatateam/neptunus/plugins"
	basic "github.com/gekatateam/neptunus/plugins/common/http"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	httpstats "github.com/gekatateam/neptunus/plugins/common/metrics"
	pkgtls "github.com/gekatateam/neptunus/plugins/common/tls"
)

type Httpl struct {
	*core.BaseInput `mapstructure:"-"`
	EnableMetrics   bool              `mapstructure:"enable_metrics"`
	Address         string            `mapstructure:"address"`
	ReadTimeout     time.Duration     `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration     `mapstructure:"write_timeout"`
	MaxConnections  int               `mapstructure:"max_connections"`
	WaitForDelivery bool              `mapstructure:"wait_for_delivery"`
	LabelHeaders    map[string]string `mapstructure:"labelheaders"`

	*basic.BasicAuth        `mapstructure:",squash"`
	*ider.Ider              `mapstructure:",squash"`
	*pkgtls.TLSServerConfig `mapstructure:",squash"`

	server   *http.Server
	listener net.Listener

	parser core.Parser
}

func (i *Httpl) Init() error {
	if len(i.Address) == 0 {
		return errors.New("address required")
	}

	if err := i.Ider.Init(); err != nil {
		return err
	}

	tlsConfig, err := i.TLSServerConfig.Config()
	if err != nil {
		return err
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
		handler = httpstats.HttpServerMiddleware(i.Pipeline, i.Alias, false, handler)
	}

	mux.Handle("/", handler)

	i.server = &http.Server{
		ReadTimeout:  i.ReadTimeout,
		WriteTimeout: i.WriteTimeout,
		Handler:      mux,
		TLSConfig:    tlsConfig,
		ErrorLog:     slog.NewLogLogger(i.Log.Handler(), slog.LevelError),
	}

	return nil
}

func (i *Httpl) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Httpl) Run() {
	i.Log.Info(fmt.Sprintf("starting http server on %v", i.Address))
	if err := i.server.Serve(i.listener); err != nil && err != http.ErrServerClosed {
		i.Log.Error("http server startup failed",
			"error", err.Error(),
		)
	} else {
		i.Log.Info("stopping http server")
	}
}

func (i *Httpl) Close() error {
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

func (i *Httpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodPut:
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	i.Log.Debug("request received",
		"sender", r.RemoteAddr,
		"path", r.URL.Path,
	)

	wg := &sync.WaitGroup{}

	var cursor, events = 0, 0
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		now := time.Now()
		cursor++

		if err := scanner.Err(); err != nil {
			i.Log.Error(fmt.Sprintf("reading error at line %v", cursor),
				"error", err,
			)
			http.Error(w, fmt.Sprintf("reading error at line %v: %v", cursor, err.Error()), http.StatusInternalServerError)
			i.Observe(metrics.EventFailed, time.Since(now))
			return
		}

		e, err := i.parser.Parse(scanner.Bytes(), r.URL.Path)
		if err != nil {
			i.Log.Error(fmt.Sprintf("parser error at line %v", cursor),
				"error", err,
			)
			http.Error(w, fmt.Sprintf("parser error at line %v: %v", cursor, err.Error()), http.StatusBadRequest)
			i.Observe(metrics.EventFailed, time.Since(now))
			return
		}

		for _, event := range e {
			event.SetLabel("server", i.Address)
			event.SetLabel("sender", r.RemoteAddr)
			event.SetLabel("method", r.Method)
			event.SetLabel("username", i.Username)

			for k, v := range i.LabelHeaders {
				h := r.Header.Get(v)
				if len(h) > 0 {
					event.SetLabel(k, h)
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
			events++
			i.Observe(metrics.EventAccepted, time.Since(now))
			now = time.Now()
		}
	}

	wg.Wait()

	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(fmt.Sprintf("accepted events: %v", events)))
	if err != nil {
		i.Log.Warn("all events accepted, but sending response to client failed",
			"error", err,
		)
	}
}

func init() {
	plugins.AddInput("httpl", func() core.Input {
		return &Httpl{
			Address:         ":9800",
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			MaxConnections:  0,
			BasicAuth:       &basic.BasicAuth{},
			Ider:            &ider.Ider{},
			TLSServerConfig: &pkgtls.TLSServerConfig{},
		}
	})
}
