package config

import (
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"

	"github.com/storeros/ipos/pkg/color"
)

type Err struct {
	msg    string
	detail string
	action string
	hint   string
}

func (u Err) Clone() Err {
	return Err{
		msg:    u.msg,
		detail: u.detail,
		action: u.action,
		hint:   u.hint,
	}
}

func (u Err) Error() string {
	if u.detail == "" {
		if u.msg != "" {
			return u.msg
		}
		return "<nil>"
	}
	return u.detail
}

func (u Err) Msg(m string, args ...interface{}) Err {
	e := u.Clone()
	e.msg = fmt.Sprintf(m, args...)
	return e
}

type ErrFn func(err error) Err

func newErrFn(msg, action, hint string) ErrFn {
	return func(err error) Err {
		u := Err{
			msg:    msg,
			action: action,
			hint:   hint,
		}
		if err != nil {
			u.detail = err.Error()
		}
		return u
	}
}

func ErrorToErr(err error) Err {
	if err == nil {
		return Err{}
	}

	if e, ok := err.(Err); ok {
		return e
	}

	if errors.Is(err, syscall.EADDRINUSE) {
		return ErrPortAlreadyInUse(err).Msg("Specified port is already in use")
	} else if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
		switch err.(type) {
		case *net.OpError:
			return ErrPortAccess(err).Msg("Insufficient permissions to use specified port")
		}
		return ErrNoPermissionsToAccessDirFiles(err).Msg("Insufficient permissions to access path")
	} else if errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrUnexpectedDataContent(err)
	} else {
		return Err{msg: err.Error()}
	}
}

func FmtError(introMsg string, err error, jsonFlag bool) string {
	renderedTxt := ""
	uiErr := ErrorToErr(err)
	if jsonFlag {
		if uiErr.detail != "" {
			return uiErr.msg + ": " + uiErr.detail
		}
		return uiErr.msg
	}
	introMsg += ": "
	if uiErr.msg != "" {
		introMsg += color.Bold(uiErr.msg)
	} else {
		introMsg += color.Bold(err.Error())
	}
	renderedTxt += color.Red(introMsg) + "\n"
	if uiErr.action != "" {
		renderedTxt += "> " + color.BgYellow(color.Black(uiErr.action)) + "\n"
	}
	if uiErr.hint != "" {
		renderedTxt += color.Bold("HINT:") + "\n"
		renderedTxt += "  " + uiErr.hint
	}
	return renderedTxt
}
