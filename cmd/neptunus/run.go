package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"sync"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/urfave/cli/v2"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pipeline/api"
	"github.com/gekatateam/neptunus/pipeline/service"
	"github.com/gekatateam/neptunus/server"
)

func run(cCtx *cli.Context) error {
	cfg, err := config.ReadConfig(cCtx.String("config"))
	if err != nil {
		return fmt.Errorf("error reading configuration file: %v", err.Error())
	}

	if err := logger.Init(cfg.Common); err != nil {
		return fmt.Errorf("logger initialization failed: %v", err.Error())
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	wg := &sync.WaitGroup{}

	storage, err := getStorage(&cfg.Engine)
	if err != nil {
		return fmt.Errorf("storage initialization failed: %v", err.Error())
	}

	s := service.Internal(storage, logger.Default.With(
		slog.Group("service",
			"kind", "internal",
		),
	))
	metrics.CollectPipes(s.Stats)

	restApi := api.Rest(s, logger.Default.With(
		slog.Group("controller",
			"kind", "rest",
		),
	))

	httpServer, err := server.Http(cfg.Common)
	if err != nil {
		return err
	}
	httpServer.Route("/api/v1", func(r chi.Router) {
		r.Mount("/pipelines", restApi.Router())
	})

	wg.Add(1)
	go func() {
		if err := httpServer.Serve(); err != nil {
			logger.Default.Error("http server startup failed",
				"error", err,
			)
			os.Exit(1)
		}
		wg.Done()
	}()

	if err := s.StartAll(); err != nil {
		return err
	}

	<-quit
	s.StopAll()

	if err := httpServer.Shutdown(context.Background()); err != nil {
		logger.Default.Warn("http server stopped with error",
			"error", err,
		)
	}

	if err := storage.Close(); err != nil {
		logger.Default.Warn("storage closed with error",
			"error", err,
		)
	}

	wg.Wait()
	logger.Default.Info("we're done here")

	return nil
}
