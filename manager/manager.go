package manager

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gekatateam/pipeline/config"
)

type managerServer struct {
	srv *http.Server
	l   net.Listener
}

func NewManagerServer(cfg config.Common) (*managerServer, error) {
	l, err := net.Listen("tcp", cfg.MgmtAddr)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
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

	return &managerServer{s, l}, nil
}

func (s *managerServer) Serve() error {
	if err := s.srv.Serve(s.l); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *managerServer) Shutdown(ctx context.Context) error {
	defer s.l.Close()
	s.srv.SetKeepAlivesEnabled(false)
	if err := s.srv.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
