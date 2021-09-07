// +build windows plan9 solaris

package http

import "net"

var listen = net.Listen
var fallbackListen = net.Listen
