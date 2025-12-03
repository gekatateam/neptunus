package pipeline

import (
	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/metrics"
)

type Service interface {
	Start(id string) error
	Stop(id string) error
	State(id string) (string, error, error)
	List() ([]*config.Pipeline, error)
	Get(id string) (*config.Pipeline, error)
	Add(pipe *config.Pipeline) error
	Update(pipe *config.Pipeline) error
	Delete(id string) error
}

type Storage interface {
	Acquire(id string) error
	Release(id string) error
	List() ([]*config.Pipeline, error)
	Get(id string) (*config.Pipeline, error)
	Add(pipe *config.Pipeline) error
	Update(pipe *config.Pipeline) error
	Delete(id string) error
	Close() error
}

type Stater interface {
	Stats() []metrics.PipelineStats
}
