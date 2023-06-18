package httpl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
)

type Httpl struct {
	alias        string
	pipe         string
	Address      string        `mapstructure:"address"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	server   *http.Server
	listener net.Listener

	log    logger.Logger
	out    chan<- *core.Event
	parser core.Parser
}

func (i *Httpl) Init(config map[string]any, alias, pipeline string, log logger.Logger) error {
	if err := mapstructure.Decode(config, i); err != nil {
		return err
	}

	if i.parser == nil {
		return errors.New("httpl input requires parser plugin")
	}

	if len(i.Address) == 0 {
		return errors.New("address required")
	}

	listener, err := net.Listen("tcp", i.Address)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}

	i.listener = listener
	mux := http.NewServeMux()
	mux.Handle("/", i)
	i.server = &http.Server{
		ReadTimeout:  i.ReadTimeout,
		WriteTimeout: i.WriteTimeout,
		Handler:      mux,
	}

	i.alias = alias
	i.pipe = pipeline
	i.log = log

	return nil
}

func (i *Httpl) Prepare(out chan<- *core.Event) {
	i.out = out
}

func (i *Httpl) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Httpl) Serve() {
	i.log.Infof("starting http server on %v", i.Address)
	if err := i.server.Serve(i.listener); err != nil && err != http.ErrServerClosed {
		i.log.Errorf("http server startup failed: %v", err.Error())
	} else {
		i.log.Debug("http server stopped")
	}
}

func (i *Httpl) Close() error {
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	i.server.SetKeepAlivesEnabled(false)
	if err := i.server.Shutdown(ctx); err != nil {
		i.log.Errorf("http server graceful shutdown ends with error: %v", err.Error())
	}
	return nil
}

func (i *Httpl) Alias() string {
	return i.alias
}

func (i *Httpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
	i.log.Debugf("received request from: %v", r.RemoteAddr)

	var cursor, events = 0, 0
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		now := time.Now()
		cursor++

		if err := scanner.Err(); err != nil {
			errMsg := fmt.Sprintf("reading error at line %v: %v", cursor, err.Error())
			i.log.Errorf(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			metrics.ObserveInputSummary("httpl", i.alias, i.pipe, metrics.EventFailed, time.Since(now))
			return
		}

		e, err := i.parser.Parse(scanner.Bytes(), r.URL.Path)
		if err != nil {
			errMsg := fmt.Sprintf("parsing error at line %v: %v", cursor, err.Error())
			i.log.Errorf(errMsg)
			http.Error(w, errMsg, http.StatusBadRequest)
			metrics.ObserveInputSummary("httpl", i.alias, i.pipe, metrics.EventFailed, time.Since(now))
			return
		}

		for _, event := range e {
			event.Labels = map[string]string{
				"input":  "httpl",
				"server": i.Address,
				"sender": r.RemoteAddr,
			}
			i.out <- event
			events++
			metrics.ObserveInputSummary("httpl", i.alias, i.pipe, metrics.EventAccepted, time.Since(now))
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("accepted events: %v\n", events)))
}

func init() {
	plugins.AddInput("httpl", func () core.Input {
		return &Httpl{
			Address:      ":9800",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
	})
}
