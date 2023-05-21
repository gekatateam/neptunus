package pipeline

import "github.com/gekatateam/pipeline/config"

type ManagerError struct {
	Reason string
}

type Maganer interface {
	StartAll() error
	StopAll()
	Start(id string) error
	Stop(id string)
	List() ([]*config.Pipeline, error)
	Get(id string) (*config.Pipeline, error)
	Add(pipeline *config.Pipeline) error
	Update(pipeline *config.Pipeline) error
	Delete(id string) (*config.Pipeline, error)
}
