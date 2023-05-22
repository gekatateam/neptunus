package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/logger/logrus"
	"github.com/gekatateam/pipeline/pipeline"
)

func test(cCtx *cli.Context) error {
	cfg, err := config.ReadConfig(cCtx.String("config")); if err != nil {
		return fmt.Errorf("error reading configuration file: %v", err.Error())
	}

	if err = logrus.InitializeLogger(cfg.Common); err != nil {
		return fmt.Errorf("logger initialization failed: %v", err.Error())
	}
	log = logrus.NewLogger(map[string]any{"scope": "main"})

	storage, err := getStorage(&cfg.PipeCfg); if err != nil {
		return fmt.Errorf("storage initialization failed: %v", err.Error())
	}

	pipesCfg, err := storage.List(); if err != nil {
		return fmt.Errorf("pipelines read failed: %v", err.Error())
	}

	for _, v := range pipesCfg {
		pipe := pipeline.New(v, logrus.NewLogger(map[string]any{
			"scope": "pipeline",
			"id":    v.Settings.Id,
		}))
		err = pipe.Test()
	}

	return err
}
