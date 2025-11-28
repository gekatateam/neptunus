package pipeline

import (
	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/metrics"
)

var (
	NotFoundErr   *NotFoundError
	ConflictErr   *ConflictError
	IOErr         *IOError
	ValidationErr *ValidationError
)

type NotFoundError struct{ Err error }

func (e *NotFoundError) Error() string { return e.Err.Error() }
func (e *NotFoundError) Unwrap() error { return e.Err }

type ConflictError struct{ Err error }

func (e *ConflictError) Error() string { return e.Err.Error() }
func (e *ConflictError) Unwrap() error { return e.Err }

type IOError struct{ Err error }

func (e *IOError) Error() string { return e.Err.Error() }
func (e *IOError) Unwrap() error { return e.Err }

type ValidationError struct{ Err error }

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

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

type Stater interface {
	Stats() []metrics.PipelineStats
}

type Storage interface {
	List() ([]*config.Pipeline, error)
	Get(id string) (*config.Pipeline, error)
	Add(pipe *config.Pipeline) error
	Update(pipe *config.Pipeline) error
	Delete(id string) error
	Close() error
}
