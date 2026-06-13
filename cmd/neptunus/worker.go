package main

import (
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
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

	s := service.Worker(storage, logger.Default.With(
		slog.Group("service",
			"kind", "worker",
		),
	))

}
