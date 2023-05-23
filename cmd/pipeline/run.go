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
	"github.com/gekatateam/pipeline/manager"
	"github.com/gekatateam/pipeline/pipeline/service"
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
	log = logrus.NewLogger(map[string]any{"scope": "main"})

	manager, err := manager.NewManagerServer(cfg.Common)
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
	wg.Add(1)
	go func()  {
		if err := manager.Serve(); err != nil {
			log.Fatalf("manager server startup error: %v", err.Error())
		}
		wg.Done()
	}()
	
	storage, err := getStorage(&cfg.PipeCfg); if err != nil {
		return fmt.Errorf("storage initialization failed: %v", err.Error())
	}

	s := service.NewInternalService(storage, logrus.NewLogger(map[string]any{
		"scope": "service",
		"type":  "internal",
	}))
	if err := s.StartAll(); err != nil {
		return err
	}

	<-quit
	s.StopAll()

	if err := manager.Shutdown(context.Background()); err != nil {
		log.Errorf("manager server stop error: %v", err.Error())
	}

	wg.Wait()
	log.Info("we're done here")

	return nil
}
