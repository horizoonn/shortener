package pool

import "errors"

var (
	ErrNoRows             = errors.New("no rows")
	ErrUniqueViolation    = errors.New("unique violation")
	ErrViolatesForeignKey = errors.New("violates foreign key")
	ErrUnknown            = errors.New("unknown postgres error")
)
