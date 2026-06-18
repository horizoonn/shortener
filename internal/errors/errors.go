package errors

import stderrors "errors"

var (
	ErrInvalidArgument = stderrors.New("invalid argument")
	ErrNotFound        = stderrors.New("resource not found")
	ErrCacheMiss       = stderrors.New("cache miss")
	ErrConflict        = stderrors.New("conflict")
	ErrInternal        = stderrors.New("internal error")
)
