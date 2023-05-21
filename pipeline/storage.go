package pipeline

import (
	"fmt"

	"github.com/gekatateam/pipeline/config"
)

type state string

const (
	StateNotFound  state = "pipeline not found"
	StateDuplicate state = "duplicate pipelines found"
	StateInternal  state = "internal error occured"
)

type StorageError struct {
	Reason string
	State  state
}

func (se *StorageError) Error() string {
	return fmt.Sprintf("storage error: %v; state: %v", se.Reason, se.State)
}

type Storage interface {
	List() ([]*config.Pipeline, error)
	Get(id string) (*config.Pipeline, error)
	Add(pipeline *config.Pipeline) error
	Update(pipeline *config.Pipeline) error
	Delete(id string) (*config.Pipeline, error)
}
