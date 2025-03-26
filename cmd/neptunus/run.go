package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"sync"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/urfave/cli/v2"
	"kythe.io/kythe/go/util/datasize"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pipeline/api"
	"github.com/gekatateam/neptunus/pipeline/service"
	xerrors "github.com/gekatateam/neptunus/pkg/errors"
	"github.com/gekatateam/neptunus/pkg/memory"
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

	if err := s.StartAll(); err != nil {
		var pipelineErr *xerrors.Errorlist
		// any other error means that app must die
		if !errors.As(err, &pipelineErr) {
			return err
		}

		if cfg.Engine.FailFast {
			s.StopAll() // stop already running pipelines before exit
			return err
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Serve(); err != nil {
			logger.Default.Error("http server startup failed",
				"error", err,
			)
			os.Exit(1)
		}
	}()

	<-quit
	if err := httpServer.Shutdown(context.Background()); err != nil {
		logger.Default.Warn("http server stopped with error",
			"error", err,
		)
	}

	s.StopAll()

	if err := storage.Close(); err != nil {
		logger.Default.Warn("storage closed with error",
			"error", err,
		)
	}

	wg.Wait()
	logger.Default.Info("we're done here")

	return nil
}

func SetRuntimeParameters(config *config.Runtime) error {
	if len(config.GCPercent) > 0 {
		gcPercent, err := strconv.Atoi(strings.TrimSuffix(config.GCPercent, "%"))
		if err != nil {
			return err
		}

		debug.SetGCPercent(gcPercent)
		logger.Default.Info(fmt.Sprintf("GC percent is set to %v", config.GCPercent))
	}

	if len(config.MemLimit) > 0 {
		var memLimit uint64
		if strings.HasSuffix(config.MemLimit, "%") {
			if memory.TotalMemory() == 0 {
				logger.Default.Warn("unable to set percentage memory limit on current system")
				return nil
			}

			memLimitPercent, err := strconv.ParseUint(strings.TrimSuffix(config.MemLimit, "%"), 10, 0)
			if err != nil {
				return err
			}

			memLimit = memory.TotalMemory() * memLimitPercent / 100
		} else {
			size, err := datasize.Parse(config.MemLimit)
			if err != nil {
				return err
			}

			memLimit = size.Bytes()
		}

		debug.SetMemoryLimit(int64(memLimit))
		logger.Default.Info(fmt.Sprintf("memory limit is set to %v", datasize.Size(memLimit)))
	}

	if config.MaxThreads > 0 {
		debug.SetMaxThreads(config.MaxThreads)
		logger.Default.Info(fmt.Sprintf("max threads number is set to %v", config.MaxThreads))
	}

	if config.MaxProcs > 0 {
		runtime.GOMAXPROCS(config.MaxProcs)
		logger.Default.Info(fmt.Sprintf("max procs number is set to %v", config.MaxProcs))
	}

	return nil
}
