package tagging

import (
	"fmt"
)

type Error struct {
	err  error
	code string
}

func Errorf(format, code string, a ...interface{}) error {
	return Error{err: fmt.Errorf(format, a...), code: code}
}

func (e Error) Unwrap() error { return e.err }

func (e Error) Error() string {
	if e.err == nil {
		return "tagging: cause <nil>"
	}
	return e.err.Error()
}

func (e Error) Code() string {
	if e.code == "" {
		return "BadRequest"
	}
	return e.code
}
