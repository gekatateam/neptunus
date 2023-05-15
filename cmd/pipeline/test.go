package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/logger/logrus"
	"github.com/gekatateam/pipeline/pipeline"
)

func test(cCtx *cli.Context) error {
	cfg, err := config.ReadConfig(cCtx.String("config"))
	if err != nil {
		return fmt.Errorf("error reading configuration file: %v", err.Error())
	}

	err = logrus.InitializeLogger(cfg.Common)
	if err != nil {
		return fmt.Errorf("logger initialization failed: %v", err.Error())
	}

	pipelines, err := loadPipelines(cfg.Pipes)
	if err != nil {
		return err
	}

	for i, pipeCfg := range pipelines {
		pipeline := pipeline.NewPipeline(cfg.Pipes[i].Id, pipeCfg, logrus.NewLogger(map[string]any{
			"scope": "pipeline",
			"id":    cfg.Pipes[i].Id,
		}))
		err = pipeline.Build()
		if err != nil {
			return fmt.Errorf("pipeline %v building failed: %v", cfg.Pipes[i].Id, err.Error())
		}
	}

	return nil
}
