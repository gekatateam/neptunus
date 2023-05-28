package storage

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pipeline"
)

const pathSeparator = string(os.PathSeparator)

type fileStorage struct {
	dir string
	ext string
}

func FS(dir, ext string) *fileStorage {
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
		return nil, &pipeline.IOError{Err: err}
	}

	var pipes = make([]*config.Pipeline, 0, len(files))
	for _, v := range files {
		pipe, err := readPipeline(v)
		if err != nil {
			return nil, err
		}
		pipes = append(pipes, pipe)
	}

	return pipes, nil
}

func (s *fileStorage) Get(id string) (*config.Pipeline, error) {
	if _, err := os.Stat(s.dir + id + s.ext); os.IsNotExist(err) {
		return nil, &pipeline.NotFoundError{Err: err}
	}

	return readPipeline(s.dir + id + s.ext)
}

func (s *fileStorage) Add(pipe *config.Pipeline) error {
	if _, err := os.Stat(s.dir + pipe.Settings.Id + s.ext); os.IsExist(err) {
		return &pipeline.ConflictError{Err: err}
	}

	return writePipeline(pipe, s.dir+pipe.Settings.Id+s.ext)
}

func (s *fileStorage) Update(pipe *config.Pipeline) error {
	if _, err := os.Stat(s.dir + pipe.Settings.Id + s.ext); os.IsNotExist(err) {
		return &pipeline.NotFoundError{Err: err}
	}

	return writePipeline(pipe, s.dir+pipe.Settings.Id+s.ext)
}

func (s *fileStorage) Delete(id string) error {
	if _, err := os.Stat(s.dir + id + s.ext); os.IsNotExist(err) {
		return &pipeline.NotFoundError{Err: err}
	}

	if err := os.Remove(s.dir + id + s.ext); err != nil {
		return &pipeline.IOError{Err: err}
	}

	return nil
}

func readPipeline(file string) (*config.Pipeline, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, &pipeline.IOError{Err: err}
	}

	pipe, err := config.UnmarshalPipeline(buf, filepath.Ext(file)[1:])
	if err != nil {
		return nil, &pipeline.ValidationError{Err: err}
	}

	return pipe, nil
}

func writePipeline(pipe *config.Pipeline, file string) error {
	data, err := config.MarshalPipeline(pipe, filepath.Ext(file)[1:])
	if err != nil {
		return &pipeline.ValidationError{Err: err}
	}

	err = os.WriteFile(file, data, 0)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	return nil
}
