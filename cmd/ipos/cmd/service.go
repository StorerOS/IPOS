package cmd

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

type serviceSignal int

const (
	serviceRestart serviceSignal = iota
	serviceStop
)

var globalServiceSignalCh chan serviceSignal

var GlobalServiceDoneCh <-chan struct{}

var GlobalContext context.Context

var cancelGlobalContext context.CancelFunc

func init() {
	initGlobalContext()
}

func initGlobalContext() {
	GlobalContext, cancelGlobalContext = context.WithCancel(context.Background())
	GlobalServiceDoneCh = GlobalContext.Done()
	globalServiceSignalCh = make(chan serviceSignal)
}

func restartProcess() error {
	argv0, err := exec.LookPath(os.Args[0])
	if err != nil {
		return err
	}

	return syscall.Exec(argv0, os.Args, os.Environ())
}
