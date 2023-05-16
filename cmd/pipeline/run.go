package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/logger/logrus"
	"github.com/gekatateam/pipeline/pipeline"
)

func run(cCtx *cli.Context) error {
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(cCtx.Context)
	defer cancel()
	for i, pipeCfg := range pipelines {
		pipeline := pipeline.NewPipeline(cfg.Pipes[i].Id, cfg.Pipes[i].Lines, pipeCfg, logrus.NewLogger(map[string]any{
			"scope": "pipeline",
			"id":    cfg.Pipes[i].Id,
		}))
		err = pipeline.Build()
		if err != nil {
			return fmt.Errorf("pipeline %v building failed: %v", cfg.Pipes[i].Id, err.Error())
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			pipeline.Run2(ctx)
		}()
	}

	<-quit
	cancel()
	wg.Wait()
	log.Info("we're done here")

	return nil
}
