package server

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gekatateam/neptunus/config"
)

type httpServer struct {
	*chi.Mux
	srv *http.Server
	l   net.Listener
}

func Http(cfg config.Common) (*httpServer, error) {
	l, err := net.Listen("tcp", cfg.HttpPort)
	if err != nil {
		return nil, err
	}

	mux := chi.NewRouter()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/debug/pprof/*", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// up all probes after api server startup
	// it is dangerous to make probes depending on pipelines state
	// because, yes, they are may fail at startup, 
	// but users can deploy pipelines with `run = true` setting in runtime, 
	// can stop pipelines, try to start deployed pipelines, etc.
	mux.HandleFunc("/probe", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("service started"))
	})

	s := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}

	return &httpServer{mux, s, l}, nil
}

func (s *httpServer) Serve() error {
	if err := s.srv.Serve(s.l); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *httpServer) Shutdown(ctx context.Context) error {
	defer s.l.Close()
	s.srv.SetKeepAlivesEnabled(false)
	if err := s.srv.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
