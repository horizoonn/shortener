package errors

import stderrors "errors"

var (
	ErrInvalidArgument = stderrors.New("invalid argument")
	ErrNotFound        = stderrors.New("resource not found")
	ErrConflict        = stderrors.New("conflict")
	ErrInternal        = stderrors.New("internal error")
)
