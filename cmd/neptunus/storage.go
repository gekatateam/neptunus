package main

import (
	"fmt"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pipeline"
	"github.com/gekatateam/neptunus/pipeline/storage"
)

func getStorage(cfg *config.Engine) (pipeline.Storage, error) {
	switch cfg.Storage {
	case "fs":
		log.Infof("using file system storage at %v", cfg.File.Directory)
		return storage.FS(cfg.File.Directory, cfg.File.Extention), nil
	default:
		return nil, fmt.Errorf("unknown storage type in configuration: %v", cfg.Storage)
	}
}
