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
	"github.com/gekatateam/pipeline/pipeline/api"
	"github.com/gekatateam/pipeline/pipeline/service"
	"github.com/gekatateam/pipeline/server"
)

func run(cCtx *cli.Context) error {
	cfg, err := config.ReadConfig(cCtx.String("config")); if err != nil {
		return fmt.Errorf("error reading configuration file: %v", err.Error())
	}

	err = logrus.InitializeLogger(cfg.Common); if err != nil {
		return fmt.Errorf("logger initialization failed: %v", err.Error())
	}
	log = logrus.NewLogger(map[string]any{"scope": "main"})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	wg := &sync.WaitGroup{}

	storage, err := getStorage(&cfg.PipeCfg); if err != nil {
		return fmt.Errorf("storage initialization failed: %v", err.Error())
	}

	s := service.Internal(storage, logrus.NewLogger(map[string]any{
		"scope": "service",
		"type":  "internal",
	}))

	restApi := api.Rest(s, logrus.NewLogger(map[string]any{
		"scope": "api",
		"type":  "rest",
	}))

	httpServer, err := server.Http(cfg.Common); if err != nil {
		return err
	}
	httpServer.Mount("/api/v1", restApi.Router())

	wg.Add(1)
	go func() {
		if err := httpServer.Serve(); err != nil {
			log.Fatalf("http server startup error: %v", err.Error())
		}
		wg.Done()
	}()

	if err := s.StartAll(); err != nil {
		return err
	}

	<-quit
	s.StopAll()

	if err := httpServer.Shutdown(context.Background()); err != nil {
		log.Errorf("http server stop error: %v", err.Error())
	}

	wg.Wait()
	log.Info("we're done here")

	return nil
}
