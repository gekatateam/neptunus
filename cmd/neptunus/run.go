package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"sync"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/urfave/cli/v2"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pipeline/api"
	"github.com/gekatateam/neptunus/pipeline/service"
	xerrors "github.com/gekatateam/neptunus/pkg/errors"
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

	if err := SetRuntimeParameters(&cfg.Runtime); err != nil {
		return fmt.Errorf("runtime params set error: %w", err)
	}

	ctx, stop := signal.NotifyContext(cCtx.Context,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer stop()

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

	if err := s.StartAll(); err != nil {
		// any other error means that app must die
		if _, ok := errors.AsType[xerrors.Errorlist](err); !ok {
			return err
		}

		if cfg.Engine.FailFast {
			s.StopAll() // stop already running pipelines before exit
			return err
		}
	}

	wg.Go(func() {
		metrics.GlobalCollectorsRunner.Run(ctx, metrics.DefaultMetricCollectInterval)
	})

	wg.Go(func() {
		if err := httpServer.Serve(); err != nil {
			logger.Default.Error("http server startup failed",
				"error", err,
			)
			os.Exit(1)
		}
	})

	<-ctx.Done()
	logger.Default.Info("shutdown signal received, exiting...")
	done := make(chan struct{})
	go func() { // all shutdown logic must be here
		if err := httpServer.Shutdown(context.Background()); err != nil {
			logger.Default.Warn("http server stopped with error",
				"error", err,
			)
		}

		wg.Wait()

		s.StopAll()

		if err := storage.Close(); err != nil {
			logger.Default.Warn("storage closed with error",
				"error", err,
			)
		}

		close(done)
	}()

	select {
	case <-done:
		logger.Default.Info("we're done here")
		return nil
	case <-time.After(time.Duration(cfg.Common.GracefulTimeout) * time.Second):
		return errors.New("graceful timeout reached, shutdown forced")
	}
}
