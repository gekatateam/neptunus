package http

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/metrics"
	"github.com/gekatateam/pipeline/pkg/mapstructure"
	"github.com/gekatateam/pipeline/plugins"
)

type Http struct {
	alias        string
	Address      string        `mapstructure:"address"`
	Path         string        `mapstructure:"path"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	RoutingKey   string        `mapstructure:"routing_key"`

	server   *http.Server
	listener net.Listener

	log logger.Logger
	out chan<- *core.Event
}

func New(config map[string]any, alias string, log logger.Logger) (core.Input, error) {
	h := &Http{log: log, alias: alias}
	if err := mapstructure.Decode(config, h); err != nil {
		return nil, err
	}

	if len(h.RoutingKey) == 0 {
		return nil, errors.New("an empty routing key is not allowed")
	}

	if len(h.Path) == 0 || len(h.Address) == 0 {
		return nil, errors.New("address and path required")
	}

	listener, err := net.Listen("tcp", h.Address)
	if err != nil {
		return nil, fmt.Errorf("error creating listener: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle(h.Path, h)
	h.listener = listener
	h.server = &http.Server{
		ReadTimeout:  h.ReadTimeout,
		WriteTimeout: h.WriteTimeout,
		Handler:      mux,
	}

	return h, nil
}

func (i *Http) Init(out chan<- *core.Event) {
	i.out = out
}

func (i *Http) Serve() {
	i.log.Infof("starting http server on %v", i.Address)
	if err := i.server.Serve(i.listener); err != nil && err != http.ErrServerClosed {
		i.log.Errorf("http server startup failed: %v", err.Error())
	} else {
		i.log.Debug("http server stopped")
	}
}

func (i *Http) Close() error {
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	i.server.SetKeepAlivesEnabled(false)
	if err := i.server.Shutdown(ctx); err != nil {
		i.log.Errorf("http server graceful shutdown ends with error: %v", err.Error())
	}
	return nil
}

func (i *Http) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}

	var cursor = 0
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		now := time.Now()
		cursor++

		if err := scanner.Err(); err != nil {
			errMsg := fmt.Sprintf("reading error at line %v: %v", cursor, err.Error())
			i.log.Errorf(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			metrics.ObserveInputSummary("http", i.alias, metrics.EventFailed, time.Since(now))
			return
		}

		e := core.NewEvent(i.RoutingKey)
		err := json.Unmarshal(scanner.Bytes(), &e.Data)
		if err != nil {
			errMsg := fmt.Sprintf("bad json at line %v: %v", cursor, err.Error())
			i.log.Errorf(errMsg)
			http.Error(w, errMsg, http.StatusBadRequest)
			metrics.ObserveInputSummary("http", i.alias, metrics.EventFailed, time.Since(now))
			return
		}

		e.Labels["input"] = "http"
		e.Labels["server"] = i.Address + i.Path
		e.Labels["sender"] = r.RemoteAddr
		i.out <- e
		metrics.ObserveInputSummary("http", i.alias, metrics.EventAccepted, time.Since(now))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("accepted events: %v\n", cursor)))
}

func init() {
	plugins.AddInput("http", New)
}
