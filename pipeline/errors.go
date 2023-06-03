package pipeline

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
