// +build linux darwin dragonfly freebsd netbsd openbsd rumprun

package http

import (
	"net"

	"github.com/valyala/tcplisten"
)

var cfg = &tcplisten.Config{
	DeferAccept: true,
	FastOpen:    true,
	Backlog:     2048,
}

var listen = cfg.NewListener
var fallbackListen = net.Listen
