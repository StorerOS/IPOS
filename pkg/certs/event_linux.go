// +build linux

package certs

import "github.com/rjeczalik/notify"

var (
	eventWrite = []notify.Event{notify.InCloseWrite}
)
