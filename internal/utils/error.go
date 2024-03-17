package utils

import (
	"github.com/pkg/errors"
)

type internalErr struct {
	error
	code uint32
}

// Cause make internalErr plays nice with errors.Cause() by returning the underlying
// error
func (e *internalErr) Cause() error {
	return e.error
}

// New creates new error containing code and msg
func New(msg string, code uint32) error {
	return &internalErr{
		error: errors.New(msg),
		code:  code,
	}
}

// Wrap creates new error containing the cause, code, and msg
// if err is nil, will return nil
func Wrap(err error, msg string, code uint32) error {
	if err == nil {
		return nil
	}

	if msg == "" {
		return &internalErr{
			error: err,
			code:  code,
		}
	}

	return &internalErr{
		error: errors.Wrap(err, msg),
		code:  code,
	}
}

// Code returns the error code of the given error.
func Code(err error) uint32 {
	if err == nil {
		return 0
	}

	if iErr, ok := err.(*internalErr); ok {
		return iErr.code
	}

	type causer interface {
		Cause() error
	}

	cause, ok := err.(causer)
	if !ok {
		return 10000
	}
	return Code(cause.Cause())
}
