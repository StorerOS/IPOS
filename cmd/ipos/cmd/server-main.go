package cmd

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/storeros/ipos/cmd/ipos/config"
	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/certs"
	"github.com/storeros/ipos/pkg/cli"
	"github.com/storeros/ipos/pkg/env"
)

var ServerFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "address",
		Value: ":" + globalIPOSDefaultPort,
		Usage: "bind to a specific ADDRESS:PORT, ADDRESS can be an IP or hostname",
	},
}

var serverCmd = cli.Command{
	Name:   "server",
	Usage:  "start object storage server",
	Flags:  append(ServerFlags, GlobalFlags...),
	Action: serverMain,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} {{if .VisibleFlags}}[FLAGS] {{end}}DIR1 [DIR2..]
  {{.HelpName}} {{if .VisibleFlags}}[FLAGS] {{end}}DIR{1...64}
  {{.HelpName}} {{if .VisibleFlags}}[FLAGS] {{end}}DIR{1...64} DIR{65...128}

DIR:
  DIR points to a directory on a filesystem. When you want to combine
  multiple drives into a single large system, pass one directory per
  filesystem separated by space. You may also use a '...' convention
  to abbreviate the directory arguments. Remote directories in a
  distributed setup are encoded as HTTP(s) URIs.
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}

EXAMPLES:
  1. Start ipos server on "/home/shared" directory.
     {{.Prompt}} {{.HelpName}} /home/shared

  2. Start single node server with 64 local drives "/mnt/data1" to "/mnt/data64".
     {{.Prompt}} {{.HelpName}} /mnt/data{1...64}

  3. Start distributed ipos server on an 32 node setup with 32 drives each, run following command on all the nodes
     {{.Prompt}} {{.EnvVarSetCommand}} IPOS_ACCESS_KEY{{.AssignmentOperator}}ipos
     {{.Prompt}} {{.EnvVarSetCommand}} IPOS_SECRET_KEY{{.AssignmentOperator}}iposstorage
     {{.Prompt}} {{.HelpName}} http://node{1...32}.example.com/mnt/export{1...32}

  4. Start distributed ipos server in an expanded setup, run the following command on all the nodes
     {{.Prompt}} {{.EnvVarSetCommand}} IPOS_ACCESS_KEY{{.AssignmentOperator}}ipos
     {{.Prompt}} {{.EnvVarSetCommand}} IPOS_SECRET_KEY{{.AssignmentOperator}}iposstorage
     {{.Prompt}} {{.HelpName}} http://node{1...16}.example.com/mnt/export{1...32} \
            http://node{17...64}.example.com/mnt/export{1...64}

`,
}

func endpointsPresent(ctx *cli.Context) bool {
	endpoints := env.Get(config.EnvEndpoints, strings.Join(ctx.Args(), config.ValueSeparator))
	return len(endpoints) != 0
}

func serverHandleCmdArgs(ctx *cli.Context) {
	handleCommonCmdArgs(ctx)

	logger.FatalIf(CheckLocalServerAddr(globalCLIContext.Addr), "Unable to validate passed arguments")

	var err error

	globalIPOSAddr = globalCLIContext.Addr

	globalIPOSHost, globalIPOSPort = mustSplitHostPort(globalIPOSAddr)

	endpoints := strings.Fields(env.Get(config.EnvEndpoints, ""))
	if len(endpoints) > 0 {
		globalEndpoints, err = createServerEndpoints(endpoints...)
	} else {
		globalEndpoints, err = createServerEndpoints(ctx.Args()...)
	}
	logger.FatalIf(err, "Invalid command line arguments")

	logger.FatalIf(checkPortAvailability(globalIPOSHost, globalIPOSPort), "Unable to start the server")
}

func serverHandleEnvVars() {
	handleCommonEnvVars()
}

func newAllSubsystems() {
	globalPolicySys = NewPolicySys()
}

func serverMain(ctx *cli.Context) {
	if ctx.Args().First() == "help" || !endpointsPresent(ctx) {
		cli.ShowCommandHelpAndExit(ctx, "server", 1)
	}

	signal.Notify(globalOSSignalCh, os.Interrupt, syscall.SIGTERM)

	serverHandleCmdArgs(ctx)

	serverHandleEnvVars()

	var err error
	var handler http.Handler
	handler, err = configureServerHandler()
	if err != nil {
		logger.Fatal(err, "Unable to configure one of server's RPC services")
	}

	var getCert certs.GetCertificateFunc
	httpServer := xhttp.NewServer([]string{globalIPOSAddr}, criticalErrorHandler{handler}, getCert)
	httpServer.BaseContext = func(listener net.Listener) context.Context {
		return GlobalContext
	}
	go func() {
		globalHTTPServerErrorCh <- httpServer.Start()
	}()

	globalObjLayerMutex.Lock()
	globalHTTPServer = httpServer
	globalObjLayerMutex.Unlock()

	newObject, err := newObjectLayer(globalEndpoints)

	globalObjLayerMutex.Lock()
	globalObjectAPI = newObject
	globalObjLayerMutex.Unlock()

	newAllSubsystems()

	printStartupMessage(getAPIEndpoints())

	handleSignals()
}

func newObjectLayer(endpoints Endpoints) (newObject ObjectLayer, err error) {
	ep := endpoints[0]
	ep.Path = ""
	ep.RawQuery = ""
	ep.Fragment = ""
	return NewIPFSObjectLayer(ep.String())
}
