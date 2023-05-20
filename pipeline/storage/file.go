package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"

	"github.com/gekatateam/pipeline/config"
)

const pathSeparator = string(os.PathSeparator)

type fileStorage struct {
	dir string
}

func NewFileStorage(dir string) *fileStorage {
	if strings.HasSuffix(dir, pathSeparator) {
		dir = dir[:len(dir) - 1]
	}
	return &fileStorage{dir: dir}
}

func (s *fileStorage) List() ([]*config.Pipeline, error) {
	dirEntry, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var pipes = make([]*config.Pipeline, 0, len(dirEntry))
	for _, v := range dirEntry {
		if v.Type().IsRegular() {
			pipe, err := readPipeline(s.dir + pathSeparator + v.Name())
			if err != nil {
				return nil, err
			}
			pipes = append(pipes, pipe)
		}
	}

	return pipes, nil
}

func (s *fileStorage) Get(id string) (*config.Pipeline, error)
func (s *fileStorage) Add(pipeline *config.Pipeline) error
func (s *fileStorage) Update(pipeline *config.Pipeline) error
func (s *fileStorage) Delete(id string) (*config.Pipeline, error)

func readPipeline(file string) (*config.Pipeline, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	pipeline := config.Pipeline{}

	switch e := filepath.Ext(file); e {
	case ".toml":
		if err := toml.Unmarshal(buf, &pipeline); err != nil {
			return &pipeline, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(buf, &pipeline); err != nil {
			return &pipeline, err
		}
	case ".json":
		if err := json.Unmarshal(buf, &pipeline); err != nil {
			return &pipeline, err
		}
	default:
		return &pipeline, fmt.Errorf("unknown pipeline file extention: %v", e)
	}

	return &pipeline, nil
}
