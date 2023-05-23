package pipeline

import "fmt"

var (
	NotFoundErr   *NotFoundError
	ConflictErr   *ConflictError
	IOErr         *IOError
	ValidationErr *ValidationError

)

type NotFoundError struct{ Err error }

func (e *NotFoundError) Error() string { return fmt.Sprintf("pipeline not found: %v", e.Err) }
func (e *NotFoundError) Unwrap() error { return e.Err }

type ConflictError struct{ Err error }

func (e *ConflictError) Error() string { return fmt.Sprintf("pipeline conflict error: %v", e.Err) }
func (e *ConflictError) Unwrap() error { return e.Err }

type IOError struct{ Err error }

func (e *IOError) Error() string { return fmt.Sprintf("pipeline I/O error: %v", e.Err) }
func (e *IOError) Unwrap() error { return e.Err }

type ValidationError struct{ Err error }

func (e *ValidationError) Error() string { return fmt.Sprintf("pipeline validation error: %v", e.Err) }
func (e *ValidationError) Unwrap() error { return e.Err }
