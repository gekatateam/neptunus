package manager

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gekatateam/pipeline/config"
)

type httpServer struct {
	*chi.Mux
	srv *http.Server
	l   net.Listener
}

func NewHttpServer(cfg config.Common) (*httpServer, error) {
	l, err := net.Listen("tcp", cfg.MgmtAddr)
	if err != nil {
		return nil, err
	}

	mux := chi.NewRouter()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/debug/pprof", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
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
