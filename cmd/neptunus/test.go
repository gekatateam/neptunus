package main

import (
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/pipeline"
)

func test(cCtx *cli.Context) error {
	cfg, err := config.ReadConfig(cCtx.String("config"))
	if err != nil {
		return fmt.Errorf("error reading configuration file: %v", err.Error())
	}

	if err = logger.Init(cfg.Common); err != nil {
		return fmt.Errorf("logger initialization failed: %v", err.Error())
	}

	storage, err := getStorage(&cfg.Engine)
	if err != nil {
		return fmt.Errorf("storage initialization failed: %v", err.Error())
	}

	pipesCfg, err := storage.List()
	if err != nil {
		return fmt.Errorf("pipelines read failed: %v", err.Error())
	}

	for _, v := range pipesCfg {
		pipe := pipeline.New(v, logger.Default.With(
			slog.Group("pipeline",
				"id", v.Settings.Id,
			),
		))
		err = pipe.Test()
	}

	return err
}
