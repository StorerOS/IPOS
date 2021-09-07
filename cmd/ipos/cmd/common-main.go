package cmd

import (
	"github.com/storeros/ipos/cmd/ipos/config"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/auth"
	"github.com/storeros/ipos/pkg/cli"
	"github.com/storeros/ipos/pkg/env"
)

func init() {
	logger.Init("", "")
	logger.RegisterError(config.FmtError)

	globalConsoleSys = NewConsoleLogger(GlobalContext)
	logger.AddTarget(globalConsoleSys)
}

func handleCommonCmdArgs(ctx *cli.Context) {
	globalCLIContext.JSON = ctx.IsSet("json") || ctx.GlobalIsSet("json")
	if globalCLIContext.JSON {
		logger.EnableJSON()
	}

	globalCLIContext.Quiet = ctx.IsSet("quiet") || ctx.GlobalIsSet("quiet")
	if globalCLIContext.Quiet {
		logger.EnableQuiet()
	}

	globalCLIContext.Anonymous = ctx.IsSet("anonymous") || ctx.GlobalIsSet("anonymous")
	if globalCLIContext.Anonymous {
		logger.EnableAnonymous()
	}

	globalCLIContext.Addr = ctx.GlobalString("address")
	if globalCLIContext.Addr == "" || globalCLIContext.Addr == ":"+globalIPOSDefaultPort {
		globalCLIContext.Addr = ctx.String("address")
	}

	globalCLIContext.StrictS3Compat = true
	if ctx.IsSet("no-compat") || ctx.GlobalIsSet("no-compat") {
		globalCLIContext.StrictS3Compat = false
	}
}

func handleCommonEnvVars() {
	if env.IsSet("IPOS_ACCESS_KEY") || env.IsSet("IPOS_SECRET_KEY") {
		cred, err := auth.CreateCredentials(env.Get("IPOS_ACCESS_KEY", ""), env.Get("IPOS_SECRET_KEY", ""))
		if err != nil {
			logger.Fatal(err, "Unable to validate credentials inherited from the shell environment")
		}
		globalActiveCred = cred
		globalConfigEncrypted = true
	}
}

func logStartupMessage(msg string) {
	if globalConsoleSys != nil {
		globalConsoleSys.Send(msg, string(logger.All))
	}
	logger.StartupMessage(msg)
}
