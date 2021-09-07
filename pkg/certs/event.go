package certs

import (
	"github.com/rjeczalik/notify"
)

func isWriteEvent(event notify.Event) bool {
	for _, ev := range eventWrite {
		if event&ev != 0 {
			return true
		}
	}
	return false
}
