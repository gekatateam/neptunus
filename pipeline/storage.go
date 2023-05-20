package pipeline

import "github.com/gekatateam/pipeline/config"



type Storage interface {
	List() ([]*config.Pipeline, error)
	Get(id string) (*config.Pipeline, error)
	Add(pipeline *config.Pipeline) error
	Update(pipeline *config.Pipeline) error
	Delete(id string) (*config.Pipeline, error)