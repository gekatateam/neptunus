package pipeline

import "github.com/gekatateam/pipeline/config"

type Service interface {
	Start(id string) error
	Stop(id string) error
	State(id string) (string, error)
	List() ([]*config.Pipeline, error)
	Get(id string) (*config.Pipeline, error)
	Add(pipe *config.Pipeline) error
	Update(pipe *config.Pipeline) error
	Delete(id string) error
}