package main

import (
	"errors"
	"fmt"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/pkg/slices"

	_ "github.com/gekatateam/pipeline/plugins/filters"
	_ "github.com/gekatateam/pipeline/plugins/inputs"
	_ "github.com/gekatateam/pipeline/plugins/outputs"
	_ "github.com/gekatateam/pipeline/plugins/processors"
)

func loadPipelines(cfg []config.PipeCfg) ([]*config.Pipeline, error) {
	if len(cfg) == 0 {
		return nil, errors.New("no pipelines in configuration file provided")
	}

	var pipesIds []string
	var pipelines []*config.Pipeline
	for _, pipeCfg := range cfg {
		if len(pipeCfg.Id) == 0 {
			return nil, errors.New("pipeline id must be provided")
		}

		pipe, err := config.ReadPipeline(pipeCfg.Config.File)
		if err != nil {
			return nil, fmt.Errorf("error reading pipeline configuration: %v", err.Error())
		}

		if slices.Contains(pipesIds, pipeCfg.Id) {
			return nil, fmt.Errorf("duplicate pipeline id founded: %v", pipeCfg.Id)
		}
		pipesIds = append(pipesIds, pipeCfg.Id)

		pipelines = append(pipelines, pipe)
	}

	return pipelines, nil
}
