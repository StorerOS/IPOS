package cmd

import (
	"os"
	"strings"

	"github.com/storeros/ipos/cmd/ipos/logger"
)

func handleSignals() {
	exit := func(success bool) {
		if success {
			os.Exit(0)
		}

		os.Exit(1)
	}

	stopProcess := func() bool {
		return true
	}

	for {
		select {
		case osSignal := <-globalOSSignalCh:
			logger.Info("Exiting on signal: %s", strings.ToUpper(osSignal.String()))
			exit(stopProcess())
		}
	}
}
