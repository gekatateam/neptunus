package pipeline

import "github.com/gekatateam/pipeline/config"

type Storage interface {
	List() ([]*config.Pipeline, error)
	Get(id string) (*config.Pipeline, error)
	Add(pipe *config.Pipeline) error
	Update(pipe *config.Pipeline) error
	Delete(id string) (*config.Pipeline, error)
}
