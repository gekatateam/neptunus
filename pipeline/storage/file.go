package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
	toml2 "github.com/naoina/toml" // for marshal only

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/pipeline"
)

const pathSeparator = string(os.PathSeparator)

type fileStorage struct {
	dir string
	ext string
}

func NewFileSystem(dir, ext string) *fileStorage {
	if !strings.HasSuffix(dir, pathSeparator) {
		dir = dir + pathSeparator
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return &fileStorage{dir: dir, ext: ext}
}

func (s *fileStorage) List() ([]*config.Pipeline, error) {
	files, err := filepath.Glob(s.dir + "*" + s.ext) // /foo/bar/*.toml
	if err != nil {
		return nil, &pipeline.StorageError{Reason: err.Error(), State: pipeline.StateInternal}
	}

	var pipes = make([]*config.Pipeline, 0, len(files))
	for _, v := range files {
		pipe, err := readPipeline(v)
		if err != nil {
			return nil, &pipeline.StorageError{Reason: err.Error(), State: pipeline.StateInternal}
		}
		pipes = append(pipes, pipe)
	}

	return pipes, nil
}

func (s *fileStorage) Get(id string) (*config.Pipeline, error) {
	if _, err := os.Stat(s.dir + id + s.ext); os.IsNotExist(err) {
		return nil, &pipeline.StorageError{
			Reason: fmt.Sprintf("file %v not found", s.dir + id + s.ext), 
			State: pipeline.StateNotFound,
		}
	}

	pipe, err := readPipeline(s.dir + id + s.ext)
	if err != nil {
		return nil, &pipeline.StorageError{Reason: err.Error(), State: pipeline.StateInternal}
	}
	
	pipe.Settings.Id = id
	return pipe, nil
}

func (s *fileStorage) Add(pipe *config.Pipeline) error {
	if _, err := os.Stat(s.dir + pipe.Settings.Id + s.ext); os.IsExist(err) {
		return &pipeline.StorageError{
			Reason: fmt.Sprintf("file %v already exists", s.dir + pipe.Settings.Id + s.ext), 
			State: pipeline.StateDuplicate,
		}
	}

	if err := savePipeline(pipe, s.dir + pipe.Settings.Id + s.ext); err != nil {
		return &pipeline.StorageError{Reason: err.Error(), State: pipeline.StateInternal}
	}
	return nil
}

func (s *fileStorage) Update(pipe *config.Pipeline) error {
	if _, err := os.Stat(s.dir + pipe.Settings.Id + s.ext); os.IsNotExist(err) {
		return &pipeline.StorageError{
			Reason: fmt.Sprintf("file %v not found", s.dir + pipe.Settings.Id + s.ext), 
			State: pipeline.StateNotFound,
		}
	}

	if err := savePipeline(pipe, s.dir + pipe.Settings.Id + s.ext); err != nil {
		return &pipeline.StorageError{Reason: err.Error(), State: pipeline.StateInternal}
	}
	return nil
}

func (s *fileStorage) Delete(id string) (*config.Pipeline, error) {
	if _, err := os.Stat(s.dir + id + s.ext); os.IsNotExist(err) {
		return nil, &pipeline.StorageError{
			Reason: fmt.Sprintf("file %v not found", s.dir + id + s.ext), 
			State: pipeline.StateNotFound,
		}
	}

	pipe, err := readPipeline(s.dir + id + s.ext)
	if err != nil {
		return nil, &pipeline.StorageError{Reason: err.Error(), State: pipeline.StateInternal}
	}

	os.Remove(s.dir + id + s.ext)
	return pipe, nil
}

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

func savePipeline(pipe *config.Pipeline, file string) error {
	var content = []byte{}
	var err error

	switch e := filepath.Ext(file); e {
	case ".toml":
		if content, err = toml2.Marshal(pipe); err != nil {
			return err
		}
	case ".yaml", ".yml":
		if content, err = yaml.Marshal(pipe); err != nil {
			return err
		}
	case ".json":
		if content, err = json.Marshal(pipe); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown pipeline file extention: %v", e)
	}

	return os.WriteFile(file, content, 0)
}
