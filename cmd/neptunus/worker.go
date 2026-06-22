package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pipeline/service"
)

func worker(cCtx *cli.Context) error {
	cfg, err := config.ReadConfig(cCtx.String("config"))
	if err != nil {
		return fmt.Errorf("error reading configuration file: %v", err.Error())
	}

	if err := logger.Init(cfg.Common); err != nil {
		return fmt.Errorf("logger initialization failed: %v", err.Error())
	}

	logger.Default = logger.Default.With(
		slog.Group("service",
			"kind", "worker",
		),
	)

	if err := SetRuntimeParameters(&cfg.Runtime); err != nil {
		return fmt.Errorf("runtime params set error: %w", err)
	}

	ctx, stop := signal.NotifyContext(cCtx.Context,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer stop()

	storage, err := getStorage(&cfg.Engine)
	if err != nil {
		return fmt.Errorf("storage initialization failed: %v", err.Error())
	}
	defer storage.Close()

	pipe, err := storage.Get(cCtx.String("pipeline"))
	if err != nil {
		return fmt.Errorf("error fetching pipeline: %v", err.Error())
	}

	w, err := service.Worker(pipe, logger.Default)
	if err != nil {
		return fmt.Errorf("error creating worker: %v", err.Error())
	}

	metrics.CollectPipes(w.Stats)

	wg := &sync.WaitGroup{}
	wg.Go(func() {
		metrics.GlobalCollectorsRunner.Run(ctx, metrics.DefaultMetricCollectInterval)
	})
	wg.Go(w.Run())

	<-ctx.Done()
	logger.Default.Info("shutdown signal received, exiting...")
	done := make(chan struct{})
	go func() { // all shutdown logic must be here
		w.Stop()
		wg.Wait()
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
