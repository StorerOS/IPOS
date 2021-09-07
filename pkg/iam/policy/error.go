package iampolicy

import (
	"fmt"
)

type Error struct {
	err error
}

func Errorf(format string, a ...interface{}) error {
	return Error{err: fmt.Errorf(format, a...)}
}

func (e Error) Unwrap() error { return e.err }

func (e Error) Error() string {
	if e.err == nil {
		return "iam: cause <nil>"
	}
	return e.err.Error()
}
