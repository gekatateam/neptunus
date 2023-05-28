package main

import (
	"fmt"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pipeline"
	"github.com/gekatateam/neptunus/pipeline/storage"
)

func getStorage(cfg *config.PipeCfg) (pipeline.Storage, error) {
	switch cfg.Storage {
	case "fs":
		return storage.FS(cfg.File.Directory, cfg.File.Extention), nil
	default:
		return nil, fmt.Errorf("unknown storage type in configuration: %v", cfg.Storage)
	}
}
