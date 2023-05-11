package http

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/pkg/mapstructure"
)

type Http struct {
	Address      string        `mapstructure:"address"`
	Path         string        `mapstructure:"path"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	RoutingKey   string        `mapstructure:"routing_key"`

	server   *http.Server
	listener net.Listener

	log  logger.Logger
	out  chan<- *core.Event
	stop chan struct{}
}

func New(config map[string]any, log logger.Logger) (core.Input, error) {
	h := &Http{
		log: log,
		stop: make(chan struct{}),
	}
	if err := mapstructure.Decode(config, h); err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", h.Address)
	if err != nil {
		return nil, fmt.Errorf("error creating listener: %v", err)
	}

	h.listener = listener
	h.server = &http.Server{
		ReadTimeout:  h.ReadTimeout,
		WriteTimeout: h.WriteTimeout,
		Handler:      http.NewServeMux(),
	}

	return h, nil
}

func(i *Http) Init(out chan<- *core.Event) (stop chan<- struct{}, done <-chan struct{})
func(i *Http) Serve()
func(i *Http) Close() error
