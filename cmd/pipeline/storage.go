package main

import (
	"fmt"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/pipeline"
	"github.com/gekatateam/pipeline/pipeline/storage"
)

func getStorage(cfg *config.PipeCfg) (pipeline.Storage, error) {
	switch cfg.Storage {
	case "fs":
		return storage.NewFileSystem(cfg.File.Directory, cfg.File.Extention), nil
	default:
		return nil, fmt.Errorf("unknown storage type in configuration: %v", cfg.Storage)
	}
}
