package main

import (
	"fmt"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/pipeline"
	"github.com/gekatateam/neptunus/pipeline/storage"
)

func getStorage(cfg *config.Engine) (pipeline.Storage, error) {
	switch cfg.Storage {
	case "fs":
		logger.Default.Info(fmt.Sprintf("using file system storage at %v", cfg.File.Directory))
		return storage.FS(cfg.File.Directory, cfg.File.Extension), nil
	default:
		return nil, fmt.Errorf("unknown storage type in configuration: %v", cfg.Storage)
	}
}
